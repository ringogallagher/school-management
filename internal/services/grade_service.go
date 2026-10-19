package services

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/mzviagintsev/school-management/internal/models"
)

// GradeService handles grade recording and grade reports
type GradeService struct {
	gradeRepo   GradeRepository
	studentRepo StudentRepository
	subjectRepo SubjectRepository
	access      *AccessControl
	notifier    *NotificationService
}

// NewGradeService creates a new grade service
func NewGradeService(gradeRepo GradeRepository, studentRepo StudentRepository, subjectRepo SubjectRepository, access *AccessControl, notifier *NotificationService) *GradeService {
	return &GradeService{
		gradeRepo:   gradeRepo,
		studentRepo: studentRepo,
		subjectRepo: subjectRepo,
		access:      access,
		notifier:    notifier,
	}
}

func roundTo2(v float64) float64 {
	return math.Round(v*100) / 100
}

// Marks use the five-point scale: whole numbers 1..5 (5 = excellent, 1 = very poor).
const (
	MinMark = 1
	MaxMark = 5
)

func validateMarks(m float64) error {
	if math.IsNaN(m) || m != math.Trunc(m) || m < MinMark || m > MaxMark {
		return fmt.Errorf("mark must be a whole number from %d to %d", MinMark, MaxMark)
	}
	return nil
}

// FinalMark rounds an average to a whole five-point mark (half rounds up: 4.5 -> 5).
// An average of zero (no marks) stays zero.
func FinalMark(avg float64) int {
	if avg <= 0 {
		return 0
	}
	f := int(math.Floor(avg + 0.5))
	if f < MinMark {
		return MinMark
	}
	if f > MaxMark {
		return MaxMark
	}
	return f
}

func normalizeTerm(t string) (string, error) {
	t = strings.TrimSpace(t)
	if t == "" {
		return "", fmt.Errorf("term is required (e.g. Q1, Midterm, Final)")
	}
	if len(t) > 20 {
		return "", fmt.Errorf("term must be at most 20 characters")
	}
	return t, nil
}

// Record records a new grade. Teachers may only grade students in classes they teach.
// The class is taken from the student's current enrolment.
func (gs *GradeService) Record(ctx context.Context, actor Actor, req *models.RecordGradeRequest) (*models.Grade, error) {
	if req.StudentID == 0 || req.SubjectID == 0 {
		return nil, fmt.Errorf("student_id and subject_id are required")
	}
	if err := validateMarks(req.Marks); err != nil {
		return nil, err
	}
	term, err := normalizeTerm(req.Term)
	if err != nil {
		return nil, err
	}

	student, err := gs.studentRepo.GetByID(ctx, req.StudentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get student: %w", err)
	}
	if student == nil {
		return nil, fmt.Errorf("student: %w", ErrNotFound)
	}
	subject, err := gs.subjectRepo.GetByID(ctx, req.SubjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subject: %w", err)
	}
	if subject == nil {
		return nil, fmt.Errorf("subject: %w", ErrNotFound)
	}
	if err := gs.access.CanTeachClass(ctx, actor, student.ClassID); err != nil {
		return nil, err
	}

	grade := &models.Grade{
		StudentID:  student.ID,
		SubjectID:  subject.ID,
		ClassID:    student.ClassID,
		Marks:      req.Marks,
		Term:       term,
		RecordedBy: actor.UserID,
	}
	if err := gs.gradeRepo.Create(ctx, grade); err != nil {
		return nil, fmt.Errorf("failed to record grade: %w", err)
	}

	gs.notifier.NotifyStudentAndParents(ctx, student, models.NotificationGradePosted,
		"New grade posted",
		fmt.Sprintf("%s (%s): %s", subject.Name, term, strconv.FormatFloat(grade.Marks, 'f', -1, 64)))

	return grade, nil
}

// getEditableGrade loads a grade and checks the actor may change it:
// admins always, teachers only grades they recorded themselves.
func (gs *GradeService) getEditableGrade(ctx context.Context, actor Actor, id int64) (*models.Grade, error) {
	grade, err := gs.gradeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get grade: %w", err)
	}
	if grade == nil {
		return nil, fmt.Errorf("grade: %w", ErrNotFound)
	}
	if actor.Role == models.RoleAdmin {
		return grade, nil
	}
	if actor.Role == models.RoleTeacher && grade.RecordedBy == actor.UserID {
		return grade, nil
	}
	return nil, ErrForbidden
}

// Update changes the marks and/or term of a grade
func (gs *GradeService) Update(ctx context.Context, actor Actor, id int64, req *models.UpdateGradeRequest) (*models.Grade, error) {
	if req.Marks == nil && req.Term == nil {
		return nil, fmt.Errorf("nothing to update: provide marks and/or term")
	}
	grade, err := gs.getEditableGrade(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	if req.Marks != nil {
		if err := validateMarks(*req.Marks); err != nil {
			return nil, err
		}
		grade.Marks = *req.Marks
	}
	if req.Term != nil {
		term, err := normalizeTerm(*req.Term)
		if err != nil {
			return nil, err
		}
		grade.Term = term
	}
	if err := gs.gradeRepo.Update(ctx, grade); err != nil {
		return nil, fmt.Errorf("failed to update grade: %w", err)
	}
	return grade, nil
}

// Delete removes a grade
func (gs *GradeService) Delete(ctx context.Context, actor Actor, id int64) error {
	if _, err := gs.getEditableGrade(ctx, actor, id); err != nil {
		return err
	}
	if err := gs.gradeRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete grade: %w", err)
	}
	return nil
}

// BuildGradeReport computes per-subject averages and the overall average
// (all on the five-point scale) from a list of grades.
func BuildGradeReport(studentID int64, term string, grades []*models.GradeWithDetails) *models.StudentGradeReport {
	type acc struct {
		name  string
		sum   float64
		count int
	}
	bySubject := map[int64]*acc{}
	var total float64

	for _, g := range grades {
		a, ok := bySubject[g.SubjectID]
		if !ok {
			a = &acc{name: g.SubjectName}
			bySubject[g.SubjectID] = a
		}
		a.sum += g.Marks
		a.count++
		total += g.Marks
	}

	report := &models.StudentGradeReport{
		StudentID: studentID,
		Term:      term,
		Grades:    grades,
		Subjects:  []models.SubjectAverage{},
	}
	if report.Grades == nil {
		report.Grades = []*models.GradeWithDetails{}
	}
	if len(grades) == 0 {
		return report
	}

	for id, a := range bySubject {
		avg := a.sum / float64(a.count)
		report.Subjects = append(report.Subjects, models.SubjectAverage{
			SubjectID:   id,
			SubjectName: a.name,
			Average:     roundTo2(avg),
			Final:       FinalMark(avg),
			Count:       a.count,
		})
	}
	sort.Slice(report.Subjects, func(i, j int) bool {
		return report.Subjects[i].SubjectName < report.Subjects[j].SubjectName
	})

	report.OverallAverage = roundTo2(total / float64(len(grades)))
	return report
}

// GetStudentReport returns a student's grades, subject averages and overall average (optionally for one term)
func (gs *GradeService) GetStudentReport(ctx context.Context, actor Actor, studentID int64, term string) (*models.StudentGradeReport, error) {
	student, err := gs.studentRepo.GetByID(ctx, studentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get student: %w", err)
	}
	if student == nil {
		return nil, fmt.Errorf("student: %w", ErrNotFound)
	}
	if err := gs.access.CanViewStudent(ctx, actor, student); err != nil {
		return nil, err
	}
	term = strings.TrimSpace(term)
	grades, err := gs.gradeRepo.ListByStudent(ctx, studentID, term)
	if err != nil {
		return nil, fmt.Errorf("failed to get grades: %w", err)
	}
	return BuildGradeReport(studentID, term, grades), nil
}

// GetMyReport returns the calling student's own grade report
func (gs *GradeService) GetMyReport(ctx context.Context, actor Actor, term string) (*models.StudentGradeReport, error) {
	student, err := gs.studentRepo.GetByUserID(ctx, actor.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get student profile: %w", err)
	}
	if student == nil {
		return nil, fmt.Errorf("student profile: %w", ErrNotFound)
	}
	return gs.GetStudentReport(ctx, actor, student.ID, term)
}

// GetClassSubjectGrades returns all grades of a class in one subject (optionally one term)
func (gs *GradeService) GetClassSubjectGrades(ctx context.Context, actor Actor, classID, subjectID int64, term string) ([]*models.GradeWithDetails, error) {
	if err := gs.access.CanTeachClass(ctx, actor, classID); err != nil {
		return nil, err
	}
	grades, err := gs.gradeRepo.ListByClassAndSubject(ctx, classID, subjectID, strings.TrimSpace(term))
	if err != nil {
		return nil, fmt.Errorf("failed to get class grades: %w", err)
	}
	if grades == nil {
		grades = []*models.GradeWithDetails{}
	}
	return grades, nil
}

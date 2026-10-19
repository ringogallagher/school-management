package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/mzviagintsev/school-management/internal/models"
)

// SubjectService handles subject management business logic
type SubjectService struct {
	subjectRepo SubjectRepository
	teacherRepo TeacherRepository
}

// NewSubjectService creates a new subject service
func NewSubjectService(subjectRepo SubjectRepository, teacherRepo TeacherRepository) *SubjectService {
	return &SubjectService{subjectRepo: subjectRepo, teacherRepo: teacherRepo}
}

// Create creates a subject. Codes are normalised to upper case and must be unique.
func (ss *SubjectService) Create(ctx context.Context, req *models.CreateSubjectRequest) (*models.Subject, error) {
	name := strings.TrimSpace(req.Name)
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	if name == "" || code == "" {
		return nil, fmt.Errorf("name and code are required")
	}
	if len(code) > 20 {
		return nil, fmt.Errorf("code must be at most 20 characters")
	}
	if req.Credits < 0 {
		return nil, fmt.Errorf("credits cannot be negative")
	}

	existing, err := ss.subjectRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to check subject code: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("subject with code %s already exists", code)
	}

	credits := req.Credits
	if credits == 0 {
		credits = 3
	}

	subject := &models.Subject{
		Name:        name,
		Code:        code,
		Description: req.Description,
		Credits:     credits,
	}
	if err := ss.subjectRepo.Create(ctx, subject); err != nil {
		return nil, fmt.Errorf("failed to create subject: %w", err)
	}
	return subject, nil
}

// GetByID returns a subject by ID
func (ss *SubjectService) GetByID(ctx context.Context, id int64) (*models.Subject, error) {
	subject, err := ss.subjectRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get subject: %w", err)
	}
	if subject == nil {
		return nil, fmt.Errorf("subject: %w", ErrNotFound)
	}
	return subject, nil
}

// Update updates a subject's name, description or credits
func (ss *SubjectService) Update(ctx context.Context, id int64, req *models.UpdateSubjectRequest) (*models.Subject, error) {
	subject, err := ss.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Credits < 0 {
		return nil, fmt.Errorf("credits cannot be negative")
	}
	if name := strings.TrimSpace(req.Name); name != "" {
		subject.Name = name
	}
	if req.Description != "" {
		subject.Description = req.Description
	}
	if req.Credits > 0 {
		subject.Credits = req.Credits
	}
	if err := ss.subjectRepo.Update(ctx, subject); err != nil {
		return nil, fmt.Errorf("failed to update subject: %w", err)
	}
	return subject, nil
}

// Delete deletes a subject. The database refuses (ON DELETE RESTRICT) if grades
// or timetable entries still reference it.
func (ss *SubjectService) Delete(ctx context.Context, id int64) error {
	if _, err := ss.GetByID(ctx, id); err != nil {
		return err
	}
	if err := ss.subjectRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete subject (it may still be used by grades or the timetable): %w", err)
	}
	return nil
}

// List returns a paginated list of subjects
func (ss *SubjectService) List(ctx context.Context, limit, offset int) ([]*models.Subject, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	subjects, err := ss.subjectRepo.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list subjects: %w", err)
	}
	return subjects, nil
}

// AssignToTeacher replaces the set of subjects a teacher (profile ID) is qualified to teach.
// An empty list removes all qualifications (the teacher may then teach any subject).
func (ss *SubjectService) AssignToTeacher(ctx context.Context, teacherID int64, subjectIDs []int64) ([]*models.Subject, error) {
	teacher, err := ss.teacherRepo.GetByID(ctx, teacherID)
	if err != nil {
		return nil, fmt.Errorf("failed to get teacher: %w", err)
	}
	if teacher == nil {
		return nil, fmt.Errorf("teacher: %w", ErrNotFound)
	}
	seen := map[int64]bool{}
	unique := make([]int64, 0, len(subjectIDs))
	for _, id := range subjectIDs {
		if seen[id] {
			continue
		}
		seen[id] = true
		s, err := ss.subjectRepo.GetByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to get subject: %w", err)
		}
		if s == nil {
			return nil, fmt.Errorf("subject %d: %w", id, ErrNotFound)
		}
		unique = append(unique, id)
	}
	if err := ss.teacherRepo.SetSubjects(ctx, teacherID, unique); err != nil {
		return nil, fmt.Errorf("failed to assign subjects: %w", err)
	}
	return ss.TeacherSubjects(ctx, teacherID)
}

// TeacherSubjects lists the subjects a teacher is qualified to teach
func (ss *SubjectService) TeacherSubjects(ctx context.Context, teacherID int64) ([]*models.Subject, error) {
	subjects, err := ss.teacherRepo.ListSubjects(ctx, teacherID)
	if err != nil {
		return nil, fmt.Errorf("failed to list teacher subjects: %w", err)
	}
	if subjects == nil {
		subjects = []*models.Subject{}
	}
	return subjects, nil
}

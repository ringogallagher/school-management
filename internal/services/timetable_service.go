package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mzviagintsev/school-management/internal/models"
)

// TimetableService manages the weekly timetable and prevents scheduling conflicts
type TimetableService struct {
	timetableRepo TimetableRepository
	classRepo     ClassRepository
	subjectRepo   SubjectRepository
	teacherRepo   TeacherRepository
	studentRepo   StudentRepository
}

// NewTimetableService creates a new timetable service
func NewTimetableService(timetableRepo TimetableRepository, classRepo ClassRepository, subjectRepo SubjectRepository, teacherRepo TeacherRepository, studentRepo StudentRepository) *TimetableService {
	return &TimetableService{
		timetableRepo: timetableRepo,
		classRepo:     classRepo,
		subjectRepo:   subjectRepo,
		teacherRepo:   teacherRepo,
		studentRepo:   studentRepo,
	}
}

// parseClock parses "HH:MM" (or "HH:MM:SS" as returned by PostgreSQL) into minutes since midnight
func parseClock(s string) (int, error) {
	for _, layout := range []string{"15:04", "15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Hour()*60 + t.Minute(), nil
		}
	}
	return 0, fmt.Errorf("invalid time %q, expected HH:MM", s)
}

func formatClock(minutes int) string {
	return fmt.Sprintf("%02d:%02d", minutes/60, minutes%60)
}

// overlaps reports whether [aStart, aEnd) and [bStart, bEnd) intersect.
// Back-to-back lessons (one ends 09:00, next starts 09:00) do not overlap.
func overlaps(aStart, aEnd, bStart, bEnd int) bool {
	return aStart < bEnd && bStart < aEnd
}

// FindConflicts returns a human-readable description of every clash between the
// candidate entry and the existing entries of the same day. Three resources can
// clash: the class (cannot attend two lessons at once), the teacher, and the room.
// excludeID skips the entry being updated.
func FindConflicts(candidate *models.Timetable, sameDay []*models.Timetable, excludeID int64) ([]string, error) {
	cStart, err := parseClock(candidate.StartTime)
	if err != nil {
		return nil, err
	}
	cEnd, err := parseClock(candidate.EndTime)
	if err != nil {
		return nil, err
	}

	var conflicts []string
	for _, e := range sameDay {
		if e.ID == excludeID || e.DayOfWeek != candidate.DayOfWeek {
			continue
		}
		eStart, err := parseClock(e.StartTime)
		if err != nil {
			return nil, err
		}
		eEnd, err := parseClock(e.EndTime)
		if err != nil {
			return nil, err
		}
		if !overlaps(cStart, cEnd, eStart, eEnd) {
			continue
		}
		slot := fmt.Sprintf("%s-%s", formatClock(eStart), formatClock(eEnd))
		if e.ClassID == candidate.ClassID {
			conflicts = append(conflicts, "the class already has a lesson at "+slot)
		}
		if e.TeacherID == candidate.TeacherID {
			conflicts = append(conflicts, "the teacher already teaches another lesson at "+slot)
		}
		if candidate.RoomNumber != "" && strings.EqualFold(e.RoomNumber, candidate.RoomNumber) {
			conflicts = append(conflicts, fmt.Sprintf("room %s is already in use at %s", candidate.RoomNumber, slot))
		}
	}
	return conflicts, nil
}

// buildEntry validates a request and turns it into a timetable entry
func (ts *TimetableService) buildEntry(ctx context.Context, req *models.TimetableRequest) (*models.Timetable, error) {
	if req.ClassID == 0 || req.SubjectID == 0 || req.TeacherID == 0 {
		return nil, fmt.Errorf("class_id, subject_id and teacher_id are required")
	}
	if req.DayOfWeek < 0 || req.DayOfWeek > 6 {
		return nil, fmt.Errorf("day_of_week must be between 0 (Sunday) and 6 (Saturday)")
	}
	start, err := parseClock(req.StartTime)
	if err != nil {
		return nil, err
	}
	end, err := parseClock(req.EndTime)
	if err != nil {
		return nil, err
	}
	if end <= start {
		return nil, fmt.Errorf("end_time must be after start_time")
	}

	class, err := ts.classRepo.GetByID(ctx, req.ClassID)
	if err != nil {
		return nil, fmt.Errorf("failed to get class: %w", err)
	}
	if class == nil {
		return nil, fmt.Errorf("class: %w", ErrNotFound)
	}
	subject, err := ts.subjectRepo.GetByID(ctx, req.SubjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subject: %w", err)
	}
	if subject == nil {
		return nil, fmt.Errorf("subject: %w", ErrNotFound)
	}
	teacher, err := ts.teacherRepo.GetByID(ctx, req.TeacherID)
	if err != nil {
		return nil, fmt.Errorf("failed to get teacher: %w", err)
	}
	if teacher == nil {
		return nil, fmt.Errorf("teacher: %w", ErrNotFound)
	}

	// If the teacher has assigned subjects, they may only be scheduled for those.
	// Teachers without any assignment are unrestricted (keeps setup simple for small schools).
	qualified, err := ts.teacherRepo.ListSubjects(ctx, req.TeacherID)
	if err != nil {
		return nil, fmt.Errorf("failed to get teacher subjects: %w", err)
	}
	if len(qualified) > 0 {
		ok := false
		for _, q := range qualified {
			if q.ID == req.SubjectID {
				ok = true
				break
			}
		}
		if !ok {
			return nil, fmt.Errorf("teacher is not assigned to teach %s", subject.Name)
		}
	}

	return &models.Timetable{
		ClassID:    req.ClassID,
		SubjectID:  req.SubjectID,
		TeacherID:  req.TeacherID,
		DayOfWeek:  req.DayOfWeek,
		StartTime:  formatClock(start),
		EndTime:    formatClock(end),
		RoomNumber: strings.TrimSpace(req.RoomNumber),
	}, nil
}

func (ts *TimetableService) checkConflicts(ctx context.Context, entry *models.Timetable, excludeID int64) error {
	sameDay, err := ts.timetableRepo.ListByDay(ctx, entry.DayOfWeek)
	if err != nil {
		return fmt.Errorf("failed to load timetable: %w", err)
	}
	conflicts, err := FindConflicts(entry, sameDay, excludeID)
	if err != nil {
		return err
	}
	if len(conflicts) > 0 {
		return fmt.Errorf("scheduling conflict: %s", strings.Join(conflicts, "; "))
	}
	return nil
}

// Create adds a timetable entry after checking for class, teacher and room conflicts
func (ts *TimetableService) Create(ctx context.Context, req *models.TimetableRequest) (*models.Timetable, error) {
	entry, err := ts.buildEntry(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := ts.checkConflicts(ctx, entry, 0); err != nil {
		return nil, err
	}
	if err := ts.timetableRepo.Create(ctx, entry); err != nil {
		return nil, fmt.Errorf("failed to create timetable entry: %w", err)
	}
	return entry, nil
}

// Update replaces a timetable entry, re-checking conflicts (ignoring the entry itself)
func (ts *TimetableService) Update(ctx context.Context, id int64, req *models.TimetableRequest) (*models.Timetable, error) {
	existing, err := ts.timetableRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get timetable entry: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("timetable entry: %w", ErrNotFound)
	}
	entry, err := ts.buildEntry(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := ts.checkConflicts(ctx, entry, id); err != nil {
		return nil, err
	}
	entry.ID = id
	entry.CreatedAt = existing.CreatedAt
	if err := ts.timetableRepo.Update(ctx, entry); err != nil {
		return nil, fmt.Errorf("failed to update timetable entry: %w", err)
	}
	return entry, nil
}

// Delete removes a timetable entry
func (ts *TimetableService) Delete(ctx context.Context, id int64) error {
	existing, err := ts.timetableRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get timetable entry: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("timetable entry: %w", ErrNotFound)
	}
	if err := ts.timetableRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete timetable entry: %w", err)
	}
	return nil
}

// ListByClass returns the weekly timetable of a class
func (ts *TimetableService) ListByClass(ctx context.Context, classID int64) ([]*models.TimetableWithDetails, error) {
	items, err := ts.timetableRepo.ListByClass(ctx, classID)
	if err != nil {
		return nil, fmt.Errorf("failed to get class timetable: %w", err)
	}
	if items == nil {
		items = []*models.TimetableWithDetails{}
	}
	return items, nil
}

// ListByTeacher returns the weekly timetable of a teacher (teacher profile ID)
func (ts *TimetableService) ListByTeacher(ctx context.Context, teacherID int64) ([]*models.TimetableWithDetails, error) {
	items, err := ts.timetableRepo.ListByTeacher(ctx, teacherID)
	if err != nil {
		return nil, fmt.Errorf("failed to get teacher timetable: %w", err)
	}
	if items == nil {
		items = []*models.TimetableWithDetails{}
	}
	return items, nil
}

// ListForActor returns "my timetable": a teacher's own lessons or a student's class timetable
func (ts *TimetableService) ListForActor(ctx context.Context, actor Actor) ([]*models.TimetableWithDetails, error) {
	switch actor.Role {
	case models.RoleTeacher:
		teacher, err := ts.teacherRepo.GetByUserID(ctx, actor.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to get teacher profile: %w", err)
		}
		if teacher == nil {
			return nil, fmt.Errorf("teacher profile: %w", ErrNotFound)
		}
		return ts.ListByTeacher(ctx, teacher.ID)
	case models.RoleStudent:
		student, err := ts.studentRepo.GetByUserID(ctx, actor.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to get student profile: %w", err)
		}
		if student == nil {
			return nil, fmt.Errorf("student profile: %w", ErrNotFound)
		}
		return ts.ListByClass(ctx, student.ClassID)
	default:
		return nil, fmt.Errorf("personal timetable is only available to teachers and students")
	}
}

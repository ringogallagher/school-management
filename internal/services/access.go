package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/mzviagintsev/school-management/internal/models"
)

// Sentinel errors that handlers translate into HTTP status codes
// (ErrForbidden -> 403, ErrNotFound -> 404). Wrap them with fmt.Errorf("...: %w", ErrX)
// to add context while keeping errors.Is working.
var (
	ErrForbidden = errors.New("forbidden")
	ErrNotFound  = errors.New("not found")
)

// Actor is the authenticated user performing an action. Services receive it
// explicitly (instead of reading the HTTP request) so that access rules can be
// unit-tested without an HTTP layer.
type Actor struct {
	UserID int64
	Role   models.UserRole
}

// AccessControl centralises the "who may see / change what" rules that are
// shared by the attendance, grade and timetable services:
//
//   - admin   : everything
//   - teacher : classes they are the homeroom teacher of, or have at least one
//     timetable lesson with, and the students in those classes
//   - student : only their own records
//   - parent  : only the records of students linked to them
type AccessControl struct {
	classRepo     ClassRepository
	teacherRepo   TeacherRepository
	parentRepo    ParentRepository
	timetableRepo TimetableRepository
}

// NewAccessControl creates the access-control helper
func NewAccessControl(classRepo ClassRepository, teacherRepo TeacherRepository, parentRepo ParentRepository, timetableRepo TimetableRepository) *AccessControl {
	return &AccessControl{
		classRepo:     classRepo,
		teacherRepo:   teacherRepo,
		parentRepo:    parentRepo,
		timetableRepo: timetableRepo,
	}
}

// CanTeachClass returns nil if the actor may record attendance and grades for the class.
func (ac *AccessControl) CanTeachClass(ctx context.Context, actor Actor, classID int64) error {
	if actor.Role == models.RoleAdmin {
		return nil
	}
	if actor.Role != models.RoleTeacher {
		return ErrForbidden
	}

	class, err := ac.classRepo.GetByID(ctx, classID)
	if err != nil {
		return fmt.Errorf("failed to get class: %w", err)
	}
	if class == nil {
		return fmt.Errorf("class: %w", ErrNotFound)
	}
	// classes.teacher_id references users.id (homeroom teacher)
	if class.TeacherID == actor.UserID {
		return nil
	}

	// timetable.teacher_id references teachers.id, so resolve the profile first
	teacher, err := ac.teacherRepo.GetByUserID(ctx, actor.UserID)
	if err != nil {
		return fmt.Errorf("failed to get teacher profile: %w", err)
	}
	if teacher == nil {
		return ErrForbidden
	}
	teaches, err := ac.timetableRepo.TeacherTeachesClass(ctx, teacher.ID, classID)
	if err != nil {
		return fmt.Errorf("failed to check timetable: %w", err)
	}
	if !teaches {
		return ErrForbidden
	}
	return nil
}

// CanViewStudent returns nil if the actor may read the student's attendance and grades.
func (ac *AccessControl) CanViewStudent(ctx context.Context, actor Actor, student *models.Student) error {
	switch actor.Role {
	case models.RoleAdmin:
		return nil
	case models.RoleTeacher:
		return ac.CanTeachClass(ctx, actor, student.ClassID)
	case models.RoleStudent:
		if student.UserID == actor.UserID {
			return nil
		}
		return ErrForbidden
	case models.RoleParent:
		ok, err := ac.parentRepo.IsParentOf(ctx, actor.UserID, student.ID)
		if err != nil {
			return fmt.Errorf("failed to check parent link: %w", err)
		}
		if ok {
			return nil
		}
		return ErrForbidden
	default:
		return ErrForbidden
	}
}

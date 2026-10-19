package services

import (
	"context"
	"fmt"

	"github.com/mzviagintsev/school-management/internal/models"
)

// ClassService handles class management business logic
type ClassService struct {
	classRepo ClassRepository
	userRepo  UserRepository
}

// NewClassService creates a new class service
func NewClassService(classRepo ClassRepository, userRepo UserRepository) *ClassService {
	return &ClassService{
		classRepo: classRepo,
		userRepo:  userRepo,
	}
}

// Create creates a new class after verifying the assigned teacher exists and has the teacher role
func (cs *ClassService) Create(ctx context.Context, req *models.CreateClassRequest) (*models.Class, error) {
	if req.Name == "" || req.Level == "" {
		return nil, fmt.Errorf("name and level are required")
	}
	if req.TeacherID == 0 {
		return nil, fmt.Errorf("teacher_id is required")
	}

	teacher, err := cs.userRepo.GetByID(ctx, req.TeacherID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify teacher: %w", err)
	}
	if teacher == nil {
		return nil, fmt.Errorf("teacher not found")
	}
	if teacher.Role != models.RoleTeacher {
		return nil, fmt.Errorf("assigned user is not a teacher")
	}

	maxStudents := req.MaxStudents
	if maxStudents <= 0 {
		maxStudents = 30
	}

	class := &models.Class{
		Name:        req.Name,
		Level:       req.Level,
		Section:     req.Section,
		TeacherID:   req.TeacherID,
		MaxStudents: maxStudents,
	}

	if err := cs.classRepo.Create(ctx, class); err != nil {
		return nil, fmt.Errorf("failed to create class: %w", err)
	}

	return class, nil
}

// GetByID returns a class by ID
func (cs *ClassService) GetByID(ctx context.Context, id int64) (*models.Class, error) {
	class, err := cs.classRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get class: %w", err)
	}
	if class == nil {
		return nil, fmt.Errorf("class not found")
	}
	return class, nil
}

// Update updates a class, re-validating the teacher assignment when it changes
func (cs *ClassService) Update(ctx context.Context, id int64, req *models.UpdateClassRequest) (*models.Class, error) {
	class, err := cs.classRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get class: %w", err)
	}
	if class == nil {
		return nil, fmt.Errorf("class not found")
	}

	if req.TeacherID != 0 && req.TeacherID != class.TeacherID {
		teacher, err := cs.userRepo.GetByID(ctx, req.TeacherID)
		if err != nil {
			return nil, fmt.Errorf("failed to verify teacher: %w", err)
		}
		if teacher == nil || teacher.Role != models.RoleTeacher {
			return nil, fmt.Errorf("assigned user is not a valid teacher")
		}
		class.TeacherID = req.TeacherID
	}
	if req.Name != "" {
		class.Name = req.Name
	}
	if req.Level != "" {
		class.Level = req.Level
	}
	if req.Section != "" {
		class.Section = req.Section
	}
	if req.MaxStudents > 0 {
		class.MaxStudents = req.MaxStudents
	}

	if err := cs.classRepo.Update(ctx, class); err != nil {
		return nil, fmt.Errorf("failed to update class: %w", err)
	}

	return class, nil
}

// Delete deletes a class
func (cs *ClassService) Delete(ctx context.Context, id int64) error {
	if err := cs.classRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete class: %w", err)
	}
	return nil
}

// List returns a paginated list of classes
func (cs *ClassService) List(ctx context.Context, limit, offset int) ([]*models.Class, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	classes, err := cs.classRepo.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list classes: %w", err)
	}
	return classes, nil
}

// GetByTeacherID returns the classes taught by a given teacher (user ID)
func (cs *ClassService) GetByTeacherID(ctx context.Context, teacherID int64) ([]*models.Class, error) {
	classes, err := cs.classRepo.GetByTeacherID(ctx, teacherID)
	if err != nil {
		return nil, fmt.Errorf("failed to get classes for teacher: %w", err)
	}
	return classes, nil
}

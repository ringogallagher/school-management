package services

import (
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/mzviagintsev/school-management/internal/models"
)

// ParentService handles parent accounts and their links to students
type ParentService struct {
	parentRepo  ParentRepository
	studentRepo StudentRepository
	userRepo    UserRepository
}

// NewParentService creates a new parent service
func NewParentService(parentRepo ParentRepository, studentRepo StudentRepository, userRepo UserRepository) *ParentService {
	return &ParentService{
		parentRepo:  parentRepo,
		studentRepo: studentRepo,
		userRepo:    userRepo,
	}
}

// Create creates a parent: a user account with role=parent plus a parent profile.
// The account is rolled back if the profile cannot be created.
func (ps *ParentService) Create(ctx context.Context, req *models.CreateParentRequest) (*models.ParentWithUser, error) {
	if req.Email == "" || req.Password == "" || req.FirstName == "" {
		return nil, fmt.Errorf("email, password, and first name are required")
	}
	if len(req.Password) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters")
	}

	existing, err := ps.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("user with this email already exists")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		Email:        req.Email,
		PasswordHash: string(hash),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Role:         models.RoleParent,
		IsActive:     true,
	}
	if err := ps.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create parent account: %w", err)
	}

	parent := &models.Parent{
		UserID:       user.ID,
		Relationship: req.Relationship,
		PhoneNumber:  req.PhoneNumber,
	}
	if err := ps.parentRepo.Create(ctx, parent); err != nil {
		_ = ps.userRepo.Delete(ctx, user.ID)
		return nil, fmt.Errorf("failed to create parent profile: %w", err)
	}

	return &models.ParentWithUser{
		Parent:    *parent,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
	}, nil
}

// LinkStudent links a parent profile to a student
func (ps *ParentService) LinkStudent(ctx context.Context, parentID, studentID int64) error {
	parent, err := ps.parentRepo.GetByID(ctx, parentID)
	if err != nil {
		return fmt.Errorf("failed to get parent: %w", err)
	}
	if parent == nil {
		return fmt.Errorf("parent: %w", ErrNotFound)
	}
	student, err := ps.studentRepo.GetByID(ctx, studentID)
	if err != nil {
		return fmt.Errorf("failed to get student: %w", err)
	}
	if student == nil {
		return fmt.Errorf("student: %w", ErrNotFound)
	}
	if err := ps.parentRepo.LinkStudent(ctx, parentID, studentID); err != nil {
		return fmt.Errorf("failed to link parent to student: %w", err)
	}
	return nil
}

// UnlinkStudent removes a parent <-> student link
func (ps *ParentService) UnlinkStudent(ctx context.Context, parentID, studentID int64) error {
	if err := ps.parentRepo.UnlinkStudent(ctx, parentID, studentID); err != nil {
		return fmt.Errorf("failed to unlink parent from student: %w", err)
	}
	return nil
}

// ListChildren returns the students linked to the given parent user
func (ps *ParentService) ListChildren(ctx context.Context, parentUserID int64) ([]*models.StudentWithUser, error) {
	children, err := ps.parentRepo.ListChildren(ctx, parentUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to list children: %w", err)
	}
	return children, nil
}

// List returns parents with account info and number of linked children
func (ps *ParentService) List(ctx context.Context, limit, offset int) ([]*models.ParentWithUser, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	parents, err := ps.parentRepo.List(ctx, limit, max(offset, 0))
	if err != nil {
		return nil, fmt.Errorf("failed to list parents: %w", err)
	}
	return parents, nil
}

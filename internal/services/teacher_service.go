package services

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/mzviagintsev/school-management/internal/models"
)

// TeacherService handles teacher management business logic
type TeacherService struct {
	teacherRepo TeacherRepository
	userRepo    UserRepository
}

// NewTeacherService creates a new teacher service
func NewTeacherService(teacherRepo TeacherRepository, userRepo UserRepository) *TeacherService {
	return &TeacherService{
		teacherRepo: teacherRepo,
		userRepo:    userRepo,
	}
}

// Create creates a new teacher: a user account with role=teacher plus a teacher profile.
// If the profile insert fails after the user account was created, the account is rolled back
// so a failed request never leaves behind an orphaned login.
func (ts *TeacherService) Create(ctx context.Context, req *models.CreateTeacherRequest) (*models.TeacherWithUser, error) {
	if req.Email == "" || req.Password == "" || req.FirstName == "" {
		return nil, fmt.Errorf("email, password, and first name are required")
	}

	existing, err := ts.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("user with this email already exists")
	}

	var hireDate *time.Time
	if req.HireDate != "" {
		parsed, err := time.Parse("2006-01-02", req.HireDate)
		if err != nil {
			return nil, fmt.Errorf("invalid hire_date format, expected YYYY-MM-DD")
		}
		hireDate = &parsed
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		Email:        req.Email,
		PasswordHash: string(passwordHash),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Role:         models.RoleTeacher,
		IsActive:     true,
	}
	if err := ts.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create teacher account: %w", err)
	}

	teacher := &models.Teacher{
		UserID:         user.ID,
		Department:     req.Department,
		Specialization: req.Specialization,
		HireDate:       hireDate,
	}

	if err := ts.teacherRepo.Create(ctx, teacher); err != nil {
		_ = ts.userRepo.Delete(ctx, user.ID)
		return nil, fmt.Errorf("failed to create teacher profile: %w", err)
	}

	return &models.TeacherWithUser{
		Teacher:   *teacher,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		IsActive:  user.IsActive,
	}, nil
}

// GetByID returns a teacher profile by ID
func (ts *TeacherService) GetByID(ctx context.Context, id int64) (*models.Teacher, error) {
	teacher, err := ts.teacherRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get teacher: %w", err)
	}
	if teacher == nil {
		return nil, fmt.Errorf("teacher not found")
	}
	return teacher, nil
}

// Update updates a teacher's department/specialization
func (ts *TeacherService) Update(ctx context.Context, id int64, req *models.UpdateTeacherRequest) (*models.Teacher, error) {
	teacher, err := ts.teacherRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get teacher: %w", err)
	}
	if teacher == nil {
		return nil, fmt.Errorf("teacher not found")
	}

	if req.Department != "" {
		teacher.Department = req.Department
	}
	if req.Specialization != "" {
		teacher.Specialization = req.Specialization
	}

	if err := ts.teacherRepo.Update(ctx, teacher); err != nil {
		return nil, fmt.Errorf("failed to update teacher: %w", err)
	}

	return teacher, nil
}

// Delete removes a teacher's profile and deactivates (soft-deletes) the underlying user account,
// preserving historical references such as classes taught or grades recorded.
func (ts *TeacherService) Delete(ctx context.Context, id int64) error {
	teacher, err := ts.teacherRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get teacher: %w", err)
	}
	if teacher == nil {
		return fmt.Errorf("teacher not found")
	}

	if err := ts.teacherRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete teacher: %w", err)
	}

	if err := ts.userRepo.Delete(ctx, teacher.UserID); err != nil {
		return fmt.Errorf("teacher profile deleted but failed to deactivate user account: %w", err)
	}

	return nil
}

// List returns a paginated list of teachers with their linked user info
func (ts *TeacherService) List(ctx context.Context, limit, offset int) ([]*models.TeacherWithUser, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	teachers, err := ts.teacherRepo.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list teachers: %w", err)
	}
	return teachers, nil
}

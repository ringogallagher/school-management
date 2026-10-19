package services

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/mzviagintsev/school-management/internal/models"
)

// StudentService handles student management business logic
type StudentService struct {
	studentRepo StudentRepository
	classRepo   ClassRepository
	userRepo    UserRepository
}

// NewStudentService creates a new student service
func NewStudentService(studentRepo StudentRepository, classRepo ClassRepository, userRepo UserRepository) *StudentService {
	return &StudentService{
		studentRepo: studentRepo,
		classRepo:   classRepo,
		userRepo:    userRepo,
	}
}

// Create creates a new student: a user account with role=student plus a student profile.
// If the profile insert fails after the user account was created, the account is rolled back
// so a failed request never leaves behind an orphaned login.
func (ss *StudentService) Create(ctx context.Context, req *models.CreateStudentRequest) (*models.StudentWithUser, error) {
	if req.Email == "" || req.Password == "" || req.FirstName == "" {
		return nil, fmt.Errorf("email, password, and first name are required")
	}
	if req.ClassID == 0 {
		return nil, fmt.Errorf("class_id is required")
	}
	if req.StudentNumber == "" {
		return nil, fmt.Errorf("student_number is required")
	}

	class, err := ss.classRepo.GetByID(ctx, req.ClassID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify class: %w", err)
	}
	if class == nil {
		return nil, fmt.Errorf("class not found")
	}

	existingUser, err := ss.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, fmt.Errorf("user with this email already exists")
	}

	existingNumber, err := ss.studentRepo.GetByStudentNumber(ctx, req.StudentNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to check student number: %w", err)
	}
	if existingNumber != nil {
		return nil, fmt.Errorf("student number already in use")
	}

	enrollmentDate := time.Now()
	if req.EnrollmentDate != "" {
		parsed, err := time.Parse("2006-01-02", req.EnrollmentDate)
		if err != nil {
			return nil, fmt.Errorf("invalid enrollment_date format, expected YYYY-MM-DD")
		}
		enrollmentDate = parsed
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
		Role:         models.RoleStudent,
		IsActive:     true,
	}
	if err := ss.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create student account: %w", err)
	}

	student := &models.Student{
		UserID:         user.ID,
		ClassID:        req.ClassID,
		EnrollmentDate: enrollmentDate,
		StudentNumber:  req.StudentNumber,
	}

	if err := ss.studentRepo.Create(ctx, student); err != nil {
		_ = ss.userRepo.Delete(ctx, user.ID)
		return nil, fmt.Errorf("failed to create student profile: %w", err)
	}

	return &models.StudentWithUser{
		Student:   *student,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		IsActive:  user.IsActive,
	}, nil
}

// GetByID returns a student profile by ID
func (ss *StudentService) GetByID(ctx context.Context, id int64) (*models.Student, error) {
	student, err := ss.studentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get student: %w", err)
	}
	if student == nil {
		return nil, fmt.Errorf("student not found")
	}
	return student, nil
}

// Update reassigns a student to a different class
func (ss *StudentService) Update(ctx context.Context, id int64, req *models.UpdateStudentRequest) (*models.Student, error) {
	student, err := ss.studentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get student: %w", err)
	}
	if student == nil {
		return nil, fmt.Errorf("student not found")
	}

	if req.ClassID != 0 {
		class, err := ss.classRepo.GetByID(ctx, req.ClassID)
		if err != nil {
			return nil, fmt.Errorf("failed to verify class: %w", err)
		}
		if class == nil {
			return nil, fmt.Errorf("class not found")
		}
		student.ClassID = req.ClassID
	}

	if err := ss.studentRepo.Update(ctx, student); err != nil {
		return nil, fmt.Errorf("failed to update student: %w", err)
	}

	return student, nil
}

// Delete removes a student's profile and deactivates (soft-deletes) the underlying user account,
// preserving historical references such as attendance and grades.
func (ss *StudentService) Delete(ctx context.Context, id int64) error {
	student, err := ss.studentRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get student: %w", err)
	}
	if student == nil {
		return fmt.Errorf("student not found")
	}

	if err := ss.studentRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete student: %w", err)
	}

	if err := ss.userRepo.Delete(ctx, student.UserID); err != nil {
		return fmt.Errorf("student profile deleted but failed to deactivate user account: %w", err)
	}

	return nil
}

// List returns a paginated list of students with their linked user info
func (ss *StudentService) List(ctx context.Context, limit, offset int) ([]*models.StudentWithUser, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	students, err := ss.studentRepo.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list students: %w", err)
	}
	return students, nil
}

// ListByClass returns a paginated list of students enrolled in a specific class
func (ss *StudentService) ListByClass(ctx context.Context, classID int64, limit, offset int) ([]*models.StudentWithUser, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	class, err := ss.classRepo.GetByID(ctx, classID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify class: %w", err)
	}
	if class == nil {
		return nil, fmt.Errorf("class not found")
	}

	students, err := ss.studentRepo.ListByClass(ctx, classID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list students by class: %w", err)
	}
	return students, nil
}

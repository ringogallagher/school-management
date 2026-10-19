package services

import (
	"context"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/mzviagintsev/school-management/internal/models"
)

// MinPasswordLength is enforced everywhere a password is set
const MinPasswordLength = 8

// UserService implements admin user management (/api/users)
type UserService struct {
	userRepo UserRepository
}

// NewUserService creates a new user service
func NewUserService(userRepo UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

// List searches users by optional role and free-text query (name or email)
func (us *UserService) List(ctx context.Context, role models.UserRole, q string, limit, offset int) ([]*models.User, error) {
	if role != "" && !isValidRole(role) {
		return nil, fmt.Errorf("invalid role %q", role)
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	users, err := us.userRepo.Search(ctx, role, strings.TrimSpace(q), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	return users, nil
}

// GetByID returns one user
func (us *UserService) GetByID(ctx context.Context, id int64) (*models.User, error) {
	user, err := us.userRepo.GetByID(ctx, id)
	if err != nil || user == nil {
		return nil, fmt.Errorf("user: %w", ErrNotFound)
	}
	return user, nil
}

// Update edits a user's name, email or active flag. Admins cannot deactivate themselves,
// which protects against locking the last administrator out.
func (us *UserService) Update(ctx context.Context, actor Actor, id int64, req *models.UpdateUserRequest) (*models.User, error) {
	user, err := us.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.FirstName != nil {
		name := strings.TrimSpace(*req.FirstName)
		if name == "" {
			return nil, fmt.Errorf("first name cannot be empty")
		}
		user.FirstName = name
	}
	if req.LastName != nil {
		user.LastName = strings.TrimSpace(*req.LastName)
	}
	if req.Email != nil {
		email := strings.TrimSpace(*req.Email)
		if _, err := mail.ParseAddress(email); err != nil {
			return nil, fmt.Errorf("invalid email address")
		}
		if !strings.EqualFold(email, user.Email) {
			existing, err := us.userRepo.GetByEmail(ctx, email)
			if err != nil {
				return nil, fmt.Errorf("failed to check email: %w", err)
			}
			if existing != nil {
				return nil, fmt.Errorf("user with this email already exists")
			}
		}
		user.Email = email
	}
	if req.IsActive != nil {
		if !*req.IsActive && id == actor.UserID {
			return nil, fmt.Errorf("you cannot deactivate your own account")
		}
		user.IsActive = *req.IsActive
	}
	if err := us.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	return user, nil
}

// ResetPassword lets an admin set a new password for any user
func (us *UserService) ResetPassword(ctx context.Context, id int64, newPassword string) error {
	if len(newPassword) < MinPasswordLength {
		return fmt.Errorf("password must be at least %d characters", MinPasswordLength)
	}
	user, err := us.GetByID(ctx, id)
	if err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	user.PasswordHash = string(hash)
	if err := us.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to reset password: %w", err)
	}
	return nil
}

// Delete soft-deletes a user. Admins cannot delete themselves.
func (us *UserService) Delete(ctx context.Context, actor Actor, id int64) error {
	if id == actor.UserID {
		return fmt.Errorf("you cannot delete your own account")
	}
	if _, err := us.GetByID(ctx, id); err != nil {
		return err
	}
	if err := us.userRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// CountByRole returns user counts per role (for the admin dashboard)
func (us *UserService) CountByRole(ctx context.Context) (map[models.UserRole]int, error) {
	return us.userRepo.CountByRole(ctx)
}

// EnsureAdmin creates the first administrator from configuration if the system has none.
// Public self-registration is disabled, so this is how a fresh installation gets its first login.
func (us *UserService) EnsureAdmin(ctx context.Context, email, password string) error {
	counts, err := us.userRepo.CountByRole(ctx)
	if err != nil {
		return fmt.Errorf("failed to count admins: %w", err)
	}
	if counts[models.RoleAdmin] > 0 {
		return nil
	}
	if email == "" || password == "" {
		slog.Warn("no administrator exists; set ADMIN_EMAIL and ADMIN_PASSWORD to create one on startup")
		return nil
	}
	if len(password) < MinPasswordLength {
		return fmt.Errorf("ADMIN_PASSWORD must be at least %d characters", MinPasswordLength)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	admin := &models.User{
		Email:        email,
		PasswordHash: string(hash),
		FirstName:    "Administrator",
		Role:         models.RoleAdmin,
		IsActive:     true,
	}
	if err := us.userRepo.Create(ctx, admin); err != nil {
		return fmt.Errorf("failed to create bootstrap admin: %w", err)
	}
	slog.Info("created bootstrap administrator", "email", email)
	return nil
}

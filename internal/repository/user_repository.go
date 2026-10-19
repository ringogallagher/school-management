package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/mzviagintsev/school-management/internal/models"
)

// UserRepository handles user database operations
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user in the database
func (ur *UserRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (email, password_hash, first_name, last_name, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	err := ur.db.QueryRowContext(ctx, query,
		user.Email,
		user.PasswordHash,
		user.FirstName,
		user.LastName,
		user.Role,
		user.IsActive,
		now,
		now,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by ID
func (ur *UserRepository) GetByID(ctx context.Context, id int64) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, role, is_active, created_at, updated_at, deleted_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`

	user := &models.User{}
	err := ur.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetByEmail retrieves a user by email
func (ur *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, role, is_active, created_at, updated_at, deleted_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`

	user := &models.User{}
	err := ur.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Return nil instead of error if user doesn't exist
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

// Update updates a user
func (ur *UserRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET email = $1, first_name = $2, last_name = $3, role = $4, is_active = $5, updated_at = $6,
		    password_hash = $7
		WHERE id = $8 AND deleted_at IS NULL
	`

	// password_hash must be written too, otherwise ChangePassword silently has no effect
	now := time.Now()
	result, err := ur.db.ExecContext(ctx, query,
		user.Email,
		user.FirstName,
		user.LastName,
		user.Role,
		user.IsActive,
		now,
		user.PasswordHash,
		user.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	user.UpdatedAt = now
	return nil
}

// Delete soft deletes a user
func (ur *UserRepository) Delete(ctx context.Context, id int64) error {
	query := `
		UPDATE users
		SET deleted_at = $1, updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL
	`

	now := time.Now()
	result, err := ur.db.ExecContext(ctx, query, now, now, id)

	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// GetByRole retrieves all users of a specific role
func (ur *UserRepository) GetByRole(ctx context.Context, role models.UserRole, limit int, offset int) ([]*models.User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, role, is_active, created_at, updated_at, deleted_at
		FROM users
		WHERE role = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := ur.db.QueryContext(ctx, query, role, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.PasswordHash,
			&user.FirstName,
			&user.LastName,
			&user.Role,
			&user.IsActive,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// Search lists users, optionally filtered by role and by a case-insensitive
// match on name or email. Soft-deleted users are excluded.
func (ur *UserRepository) Search(ctx context.Context, role models.UserRole, q string, limit, offset int) ([]*models.User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, role, is_active, created_at, updated_at, deleted_at
		FROM users
		WHERE deleted_at IS NULL
		  AND ($1 = '' OR role::text = $1)
		  AND ($2 = '' OR email ILIKE '%' || $2 || '%'
		               OR (first_name || ' ' || last_name) ILIKE '%' || $2 || '%')
		ORDER BY last_name, first_name, id
		LIMIT $3 OFFSET $4
	`
	rows, err := ur.db.QueryContext(ctx, query, string(role), q, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}
	defer rows.Close()

	users := []*models.User{}
	for rows.Next() {
		u := &models.User{}
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName, &u.Role,
			&u.IsActive, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// CountByRole returns the number of active (not deleted) users per role
func (ur *UserRepository) CountByRole(ctx context.Context) (map[models.UserRole]int, error) {
	rows, err := ur.db.QueryContext(ctx,
		`SELECT role, COUNT(*) FROM users WHERE deleted_at IS NULL GROUP BY role`)
	if err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}
	defer rows.Close()

	counts := map[models.UserRole]int{}
	for rows.Next() {
		var role models.UserRole
		var n int
		if err := rows.Scan(&role, &n); err != nil {
			return nil, fmt.Errorf("failed to scan count: %w", err)
		}
		counts[role] = n
	}
	return counts, rows.Err()
}

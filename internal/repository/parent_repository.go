package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/mzviagintsev/school-management/internal/models"
)

// ParentRepository handles parent profiles and student <-> parent links
type ParentRepository struct {
	db *sql.DB
}

// NewParentRepository creates a new parent repository
func NewParentRepository(db *sql.DB) *ParentRepository {
	return &ParentRepository{db: db}
}

// Create inserts a parent profile
func (pr *ParentRepository) Create(ctx context.Context, p *models.Parent) error {
	query := `
		INSERT INTO parents (user_id, relationship, phone_number, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4)
		RETURNING id, created_at, updated_at
	`
	if err := pr.db.QueryRowContext(ctx, query, p.UserID, p.Relationship, p.PhoneNumber, time.Now()).
		Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return fmt.Errorf("failed to create parent: %w", err)
	}
	return nil
}

func (pr *ParentRepository) getOne(ctx context.Context, where string, arg any) (*models.Parent, error) {
	query := `
		SELECT id, user_id, COALESCE(relationship, ''), COALESCE(phone_number, ''), created_at, updated_at
		FROM parents WHERE ` + where
	p := &models.Parent{}
	err := pr.db.QueryRowContext(ctx, query, arg).
		Scan(&p.ID, &p.UserID, &p.Relationship, &p.PhoneNumber, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get parent: %w", err)
	}
	return p, nil
}

// GetByID retrieves a parent profile by ID (nil, nil if not found)
func (pr *ParentRepository) GetByID(ctx context.Context, id int64) (*models.Parent, error) {
	return pr.getOne(ctx, "id = $1", id)
}

// GetByUserID retrieves a parent profile by user ID (nil, nil if not found)
func (pr *ParentRepository) GetByUserID(ctx context.Context, userID int64) (*models.Parent, error) {
	return pr.getOne(ctx, "user_id = $1", userID)
}

// LinkStudent links a parent to a student (idempotent)
func (pr *ParentRepository) LinkStudent(ctx context.Context, parentID, studentID int64) error {
	_, err := pr.db.ExecContext(ctx,
		`INSERT INTO student_parents (student_id, parent_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		studentID, parentID)
	if err != nil {
		return fmt.Errorf("failed to link parent: %w", err)
	}
	return nil
}

// UnlinkStudent removes a parent <-> student link
func (pr *ParentRepository) UnlinkStudent(ctx context.Context, parentID, studentID int64) error {
	res, err := pr.db.ExecContext(ctx,
		`DELETE FROM student_parents WHERE student_id = $1 AND parent_id = $2`, studentID, parentID)
	if err != nil {
		return fmt.Errorf("failed to unlink parent: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("link not found")
	}
	return nil
}

// IsParentOf reports whether the parent user is linked to the student
func (pr *ParentRepository) IsParentOf(ctx context.Context, parentUserID, studentID int64) (bool, error) {
	var exists bool
	err := pr.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM student_parents sp
			JOIN parents p ON p.id = sp.parent_id
			WHERE p.user_id = $1 AND sp.student_id = $2
		)`, parentUserID, studentID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check parent link: %w", err)
	}
	return exists, nil
}

// ParentUserIDsForStudent returns the user IDs of all active parents linked to a student
func (pr *ParentRepository) ParentUserIDsForStudent(ctx context.Context, studentID int64) ([]int64, error) {
	rows, err := pr.db.QueryContext(ctx, `
		SELECT p.user_id
		FROM student_parents sp
		JOIN parents p ON p.id = sp.parent_id
		JOIN users u ON u.id = p.user_id
		WHERE sp.student_id = $1 AND u.is_active AND u.deleted_at IS NULL`, studentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get parents: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan parent id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ListChildren returns the students linked to a parent user
func (pr *ParentRepository) ListChildren(ctx context.Context, parentUserID int64) ([]*models.StudentWithUser, error) {
	rows, err := pr.db.QueryContext(ctx, `
		SELECT s.id, s.user_id, s.class_id, s.enrollment_date, s.student_number, s.created_at, s.updated_at,
		       u.email, u.first_name, u.last_name, u.is_active
		FROM student_parents sp
		JOIN parents p ON p.id = sp.parent_id
		JOIN students s ON s.id = sp.student_id
		JOIN users u ON u.id = s.user_id
		WHERE p.user_id = $1 AND u.deleted_at IS NULL
		ORDER BY u.first_name`, parentUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to list children: %w", err)
	}
	defer rows.Close()

	children := []*models.StudentWithUser{}
	for rows.Next() {
		s := &models.StudentWithUser{}
		if err := rows.Scan(&s.ID, &s.UserID, &s.ClassID, &s.EnrollmentDate, &s.StudentNumber, &s.CreatedAt, &s.UpdatedAt,
			&s.Email, &s.FirstName, &s.LastName, &s.IsActive); err != nil {
			return nil, fmt.Errorf("failed to scan child: %w", err)
		}
		children = append(children, s)
	}
	return children, rows.Err()
}

// List returns parents with their account info and the number of linked children
func (pr *ParentRepository) List(ctx context.Context, limit, offset int) ([]*models.ParentWithUser, error) {
	rows, err := pr.db.QueryContext(ctx, `
		SELECT p.id, p.user_id, COALESCE(p.relationship, ''), COALESCE(p.phone_number, ''), p.created_at, p.updated_at,
		       u.email, u.first_name, u.last_name,
		       (SELECT COUNT(*) FROM student_parents sp WHERE sp.parent_id = p.id)
		FROM parents p
		JOIN users u ON u.id = p.user_id
		WHERE u.deleted_at IS NULL
		ORDER BY u.last_name, u.first_name
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list parents: %w", err)
	}
	defer rows.Close()

	result := []*models.ParentWithUser{}
	for rows.Next() {
		p := &models.ParentWithUser{}
		if err := rows.Scan(&p.ID, &p.UserID, &p.Relationship, &p.PhoneNumber, &p.CreatedAt, &p.UpdatedAt,
			&p.Email, &p.FirstName, &p.LastName, &p.ChildCount); err != nil {
			return nil, fmt.Errorf("failed to scan parent: %w", err)
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

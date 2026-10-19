package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/mzviagintsev/school-management/internal/models"
)

// ClassRepository handles class database operations
type ClassRepository struct {
	db *sql.DB
}

// NewClassRepository creates a new class repository
func NewClassRepository(db *sql.DB) *ClassRepository {
	return &ClassRepository{db: db}
}

// Create creates a new class in the database
func (cr *ClassRepository) Create(ctx context.Context, class *models.Class) error {
	query := `
		INSERT INTO classes (name, level, section, teacher_id, max_students, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	err := cr.db.QueryRowContext(ctx, query,
		class.Name,
		class.Level,
		class.Section,
		class.TeacherID,
		class.MaxStudents,
		now,
		now,
	).Scan(&class.ID, &class.CreatedAt, &class.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create class: %w", err)
	}

	return nil
}

// GetByID retrieves a class by ID
func (cr *ClassRepository) GetByID(ctx context.Context, id int64) (*models.Class, error) {
	query := `
		SELECT id, name, level, section, teacher_id, max_students, created_at, updated_at
		FROM classes
		WHERE id = $1
	`

	class := &models.Class{}
	err := cr.db.QueryRowContext(ctx, query, id).Scan(
		&class.ID,
		&class.Name,
		&class.Level,
		&class.Section,
		&class.TeacherID,
		&class.MaxStudents,
		&class.CreatedAt,
		&class.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get class: %w", err)
	}

	return class, nil
}

// Update updates a class
func (cr *ClassRepository) Update(ctx context.Context, class *models.Class) error {
	query := `
		UPDATE classes
		SET name = $1, level = $2, section = $3, teacher_id = $4, max_students = $5, updated_at = $6
		WHERE id = $7
	`

	now := time.Now()
	result, err := cr.db.ExecContext(ctx, query,
		class.Name,
		class.Level,
		class.Section,
		class.TeacherID,
		class.MaxStudents,
		now,
		class.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update class: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("class not found")
	}

	class.UpdatedAt = now
	return nil
}

// Delete deletes a class
func (cr *ClassRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM classes WHERE id = $1`

	result, err := cr.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete class: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("class not found")
	}

	return nil
}

// List retrieves classes with pagination
func (cr *ClassRepository) List(ctx context.Context, limit, offset int) ([]*models.Class, error) {
	query := `
		SELECT id, name, level, section, teacher_id, max_students, created_at, updated_at
		FROM classes
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := cr.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list classes: %w", err)
	}
	defer rows.Close()

	var classes []*models.Class
	for rows.Next() {
		class := &models.Class{}
		err := rows.Scan(
			&class.ID,
			&class.Name,
			&class.Level,
			&class.Section,
			&class.TeacherID,
			&class.MaxStudents,
			&class.CreatedAt,
			&class.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan class: %w", err)
		}
		classes = append(classes, class)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating classes: %w", err)
	}

	return classes, nil
}

// GetByTeacherID retrieves all classes taught by a given teacher (user ID)
func (cr *ClassRepository) GetByTeacherID(ctx context.Context, teacherID int64) ([]*models.Class, error) {
	query := `
		SELECT id, name, level, section, teacher_id, max_students, created_at, updated_at
		FROM classes
		WHERE teacher_id = $1
		ORDER BY created_at DESC
	`

	rows, err := cr.db.QueryContext(ctx, query, teacherID)
	if err != nil {
		return nil, fmt.Errorf("failed to list classes by teacher: %w", err)
	}
	defer rows.Close()

	var classes []*models.Class
	for rows.Next() {
		class := &models.Class{}
		err := rows.Scan(
			&class.ID,
			&class.Name,
			&class.Level,
			&class.Section,
			&class.TeacherID,
			&class.MaxStudents,
			&class.CreatedAt,
			&class.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan class: %w", err)
		}
		classes = append(classes, class)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating classes: %w", err)
	}

	return classes, nil
}

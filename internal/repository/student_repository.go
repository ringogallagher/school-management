package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/mzviagintsev/school-management/internal/models"
)

// StudentRepository handles student database operations
type StudentRepository struct {
	db *sql.DB
}

// NewStudentRepository creates a new student repository
func NewStudentRepository(db *sql.DB) *StudentRepository {
	return &StudentRepository{db: db}
}

// Create creates a new student profile linked to an existing user account
func (sr *StudentRepository) Create(ctx context.Context, student *models.Student) error {
	query := `
		INSERT INTO students (user_id, class_id, enrollment_date, student_number, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	err := sr.db.QueryRowContext(ctx, query,
		student.UserID,
		student.ClassID,
		student.EnrollmentDate,
		student.StudentNumber,
		now,
		now,
	).Scan(&student.ID, &student.CreatedAt, &student.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create student: %w", err)
	}

	return nil
}

// GetByID retrieves a student profile by its ID
func (sr *StudentRepository) GetByID(ctx context.Context, id int64) (*models.Student, error) {
	query := `
		SELECT id, user_id, class_id, enrollment_date, student_number, created_at, updated_at
		FROM students
		WHERE id = $1
	`

	student := &models.Student{}
	err := sr.db.QueryRowContext(ctx, query, id).Scan(
		&student.ID,
		&student.UserID,
		&student.ClassID,
		&student.EnrollmentDate,
		&student.StudentNumber,
		&student.CreatedAt,
		&student.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get student: %w", err)
	}

	return student, nil
}

// GetByUserID retrieves a student profile by its linked user ID
func (sr *StudentRepository) GetByUserID(ctx context.Context, userID int64) (*models.Student, error) {
	query := `
		SELECT id, user_id, class_id, enrollment_date, student_number, created_at, updated_at
		FROM students
		WHERE user_id = $1
	`

	student := &models.Student{}
	err := sr.db.QueryRowContext(ctx, query, userID).Scan(
		&student.ID,
		&student.UserID,
		&student.ClassID,
		&student.EnrollmentDate,
		&student.StudentNumber,
		&student.CreatedAt,
		&student.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get student by user id: %w", err)
	}

	return student, nil
}

// GetByStudentNumber retrieves a student profile by its student number
func (sr *StudentRepository) GetByStudentNumber(ctx context.Context, number string) (*models.Student, error) {
	query := `
		SELECT id, user_id, class_id, enrollment_date, student_number, created_at, updated_at
		FROM students
		WHERE student_number = $1
	`

	student := &models.Student{}
	err := sr.db.QueryRowContext(ctx, query, number).Scan(
		&student.ID,
		&student.UserID,
		&student.ClassID,
		&student.EnrollmentDate,
		&student.StudentNumber,
		&student.CreatedAt,
		&student.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get student by number: %w", err)
	}

	return student, nil
}

// Update updates a student profile (currently: reassigning class)
func (sr *StudentRepository) Update(ctx context.Context, student *models.Student) error {
	query := `
		UPDATE students
		SET class_id = $1, updated_at = $2
		WHERE id = $3
	`

	now := time.Now()
	result, err := sr.db.ExecContext(ctx, query, student.ClassID, now, student.ID)

	if err != nil {
		return fmt.Errorf("failed to update student: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("student not found")
	}

	student.UpdatedAt = now
	return nil
}

// Delete deletes a student profile (the linked user account is untouched here)
func (sr *StudentRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM students WHERE id = $1`

	result, err := sr.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete student: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("student not found")
	}

	return nil
}

// List retrieves students with pagination, joined with their user account info
func (sr *StudentRepository) List(ctx context.Context, limit, offset int) ([]*models.StudentWithUser, error) {
	query := `
		SELECT s.id, s.user_id, s.class_id, s.enrollment_date, s.student_number, s.created_at, s.updated_at,
		       u.email, u.first_name, u.last_name, u.is_active
		FROM students s
		JOIN users u ON u.id = s.user_id
		WHERE u.deleted_at IS NULL
		ORDER BY s.created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := sr.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list students: %w", err)
	}
	defer rows.Close()

	var students []*models.StudentWithUser
	for rows.Next() {
		s := &models.StudentWithUser{}
		err := rows.Scan(
			&s.ID,
			&s.UserID,
			&s.ClassID,
			&s.EnrollmentDate,
			&s.StudentNumber,
			&s.CreatedAt,
			&s.UpdatedAt,
			&s.Email,
			&s.FirstName,
			&s.LastName,
			&s.IsActive,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan student: %w", err)
		}
		students = append(students, s)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating students: %w", err)
	}

	return students, nil
}

// ListByClass retrieves students enrolled in a specific class, joined with their user account info
func (sr *StudentRepository) ListByClass(ctx context.Context, classID int64, limit, offset int) ([]*models.StudentWithUser, error) {
	query := `
		SELECT s.id, s.user_id, s.class_id, s.enrollment_date, s.student_number, s.created_at, s.updated_at,
		       u.email, u.first_name, u.last_name, u.is_active
		FROM students s
		JOIN users u ON u.id = s.user_id
		WHERE s.class_id = $1 AND u.deleted_at IS NULL
		ORDER BY u.last_name, u.first_name
		LIMIT $2 OFFSET $3
	`

	rows, err := sr.db.QueryContext(ctx, query, classID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list students by class: %w", err)
	}
	defer rows.Close()

	var students []*models.StudentWithUser
	for rows.Next() {
		s := &models.StudentWithUser{}
		err := rows.Scan(
			&s.ID,
			&s.UserID,
			&s.ClassID,
			&s.EnrollmentDate,
			&s.StudentNumber,
			&s.CreatedAt,
			&s.UpdatedAt,
			&s.Email,
			&s.FirstName,
			&s.LastName,
			&s.IsActive,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan student: %w", err)
		}
		students = append(students, s)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating students: %w", err)
	}

	return students, nil
}

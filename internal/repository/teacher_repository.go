package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/mzviagintsev/school-management/internal/models"
)

// TeacherRepository handles teacher database operations
type TeacherRepository struct {
	db *sql.DB
}

// NewTeacherRepository creates a new teacher repository
func NewTeacherRepository(db *sql.DB) *TeacherRepository {
	return &TeacherRepository{db: db}
}

// Create creates a new teacher profile linked to an existing user account
func (tr *TeacherRepository) Create(ctx context.Context, teacher *models.Teacher) error {
	query := `
		INSERT INTO teachers (user_id, department, specialization, hire_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	var hireDate interface{}
	if teacher.HireDate != nil {
		hireDate = *teacher.HireDate
	}

	err := tr.db.QueryRowContext(ctx, query,
		teacher.UserID,
		teacher.Department,
		teacher.Specialization,
		hireDate,
		now,
		now,
	).Scan(&teacher.ID, &teacher.CreatedAt, &teacher.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create teacher: %w", err)
	}

	return nil
}

// GetByID retrieves a teacher profile by its ID
func (tr *TeacherRepository) GetByID(ctx context.Context, id int64) (*models.Teacher, error) {
	query := `
		SELECT id, user_id, department, specialization, hire_date, created_at, updated_at
		FROM teachers
		WHERE id = $1
	`

	teacher := &models.Teacher{}
	err := tr.db.QueryRowContext(ctx, query, id).Scan(
		&teacher.ID,
		&teacher.UserID,
		&teacher.Department,
		&teacher.Specialization,
		&teacher.HireDate,
		&teacher.CreatedAt,
		&teacher.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get teacher: %w", err)
	}

	return teacher, nil
}

// GetByUserID retrieves a teacher profile by its linked user ID
func (tr *TeacherRepository) GetByUserID(ctx context.Context, userID int64) (*models.Teacher, error) {
	query := `
		SELECT id, user_id, department, specialization, hire_date, created_at, updated_at
		FROM teachers
		WHERE user_id = $1
	`

	teacher := &models.Teacher{}
	err := tr.db.QueryRowContext(ctx, query, userID).Scan(
		&teacher.ID,
		&teacher.UserID,
		&teacher.Department,
		&teacher.Specialization,
		&teacher.HireDate,
		&teacher.CreatedAt,
		&teacher.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get teacher by user id: %w", err)
	}

	return teacher, nil
}

// Update updates a teacher profile
func (tr *TeacherRepository) Update(ctx context.Context, teacher *models.Teacher) error {
	query := `
		UPDATE teachers
		SET department = $1, specialization = $2, hire_date = $3, updated_at = $4
		WHERE id = $5
	`

	now := time.Now()
	var hireDate interface{}
	if teacher.HireDate != nil {
		hireDate = *teacher.HireDate
	}

	result, err := tr.db.ExecContext(ctx, query,
		teacher.Department,
		teacher.Specialization,
		hireDate,
		now,
		teacher.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update teacher: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("teacher not found")
	}

	teacher.UpdatedAt = now
	return nil
}

// Delete deletes a teacher profile (the linked user account is untouched here)
func (tr *TeacherRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM teachers WHERE id = $1`

	result, err := tr.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete teacher: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("teacher not found")
	}

	return nil
}

// List retrieves teachers with pagination, joined with their user account info
func (tr *TeacherRepository) List(ctx context.Context, limit, offset int) ([]*models.TeacherWithUser, error) {
	query := `
		SELECT t.id, t.user_id, t.department, t.specialization, t.hire_date, t.created_at, t.updated_at,
		       u.email, u.first_name, u.last_name, u.is_active
		FROM teachers t
		JOIN users u ON u.id = t.user_id
		WHERE u.deleted_at IS NULL
		ORDER BY t.created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := tr.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list teachers: %w", err)
	}
	defer rows.Close()

	var teachers []*models.TeacherWithUser
	for rows.Next() {
		t := &models.TeacherWithUser{}
		err := rows.Scan(
			&t.ID,
			&t.UserID,
			&t.Department,
			&t.Specialization,
			&t.HireDate,
			&t.CreatedAt,
			&t.UpdatedAt,
			&t.Email,
			&t.FirstName,
			&t.LastName,
			&t.IsActive,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan teacher: %w", err)
		}
		teachers = append(teachers, t)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating teachers: %w", err)
	}

	return teachers, nil
}

// SetSubjects replaces the list of subjects a teacher is qualified to teach (atomically)
func (tr *TeacherRepository) SetSubjects(ctx context.Context, teacherID int64, subjectIDs []int64) error {
	tx, err := tr.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM teacher_subjects WHERE teacher_id = $1`, teacherID); err != nil {
		return fmt.Errorf("failed to clear teacher subjects: %w", err)
	}
	for _, sid := range subjectIDs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO teacher_subjects (teacher_id, subject_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			teacherID, sid); err != nil {
			return fmt.Errorf("failed to assign subject %d: %w", sid, err)
		}
	}
	return tx.Commit()
}

// ListSubjects returns the subjects a teacher is qualified to teach
func (tr *TeacherRepository) ListSubjects(ctx context.Context, teacherID int64) ([]*models.Subject, error) {
	rows, err := tr.db.QueryContext(ctx, `
		SELECT s.id, s.name, s.code, COALESCE(s.description, ''), s.credits, s.created_at, s.updated_at
		FROM teacher_subjects ts
		JOIN subjects s ON s.id = ts.subject_id
		WHERE ts.teacher_id = $1
		ORDER BY s.name`, teacherID)
	if err != nil {
		return nil, fmt.Errorf("failed to list teacher subjects: %w", err)
	}
	defer rows.Close()

	subjects := []*models.Subject{}
	for rows.Next() {
		s := &models.Subject{}
		if err := rows.Scan(&s.ID, &s.Name, &s.Code, &s.Description, &s.Credits, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan subject: %w", err)
		}
		subjects = append(subjects, s)
	}
	return subjects, rows.Err()
}

package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/mzviagintsev/school-management/internal/models"
)

// SubjectRepository handles subject database operations
type SubjectRepository struct {
	db *sql.DB
}

// NewSubjectRepository creates a new subject repository
func NewSubjectRepository(db *sql.DB) *SubjectRepository {
	return &SubjectRepository{db: db}
}

const subjectColumns = `id, name, code, COALESCE(description, ''), credits, created_at, updated_at`

func scanSubject(row interface{ Scan(...any) error }) (*models.Subject, error) {
	s := &models.Subject{}
	err := row.Scan(&s.ID, &s.Name, &s.Code, &s.Description, &s.Credits, &s.CreatedAt, &s.UpdatedAt)
	return s, err
}

// Create inserts a subject
func (sr *SubjectRepository) Create(ctx context.Context, s *models.Subject) error {
	query := `
		INSERT INTO subjects (name, code, description, credits, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		RETURNING id, created_at, updated_at
	`
	if err := sr.db.QueryRowContext(ctx, query, s.Name, s.Code, s.Description, s.Credits, time.Now()).
		Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return fmt.Errorf("failed to create subject: %w", err)
	}
	return nil
}

func (sr *SubjectRepository) getOne(ctx context.Context, where string, arg any) (*models.Subject, error) {
	s, err := scanSubject(sr.db.QueryRowContext(ctx, `SELECT `+subjectColumns+` FROM subjects WHERE `+where, arg))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get subject: %w", err)
	}
	return s, nil
}

// GetByID retrieves a subject by ID (nil, nil if not found)
func (sr *SubjectRepository) GetByID(ctx context.Context, id int64) (*models.Subject, error) {
	return sr.getOne(ctx, "id = $1", id)
}

// GetByCode retrieves a subject by its unique code (nil, nil if not found)
func (sr *SubjectRepository) GetByCode(ctx context.Context, code string) (*models.Subject, error) {
	return sr.getOne(ctx, "code = $1", code)
}

// Update updates a subject
func (sr *SubjectRepository) Update(ctx context.Context, s *models.Subject) error {
	now := time.Now()
	res, err := sr.db.ExecContext(ctx,
		`UPDATE subjects SET name = $1, description = $2, credits = $3, updated_at = $4 WHERE id = $5`,
		s.Name, s.Description, s.Credits, now, s.ID)
	if err != nil {
		return fmt.Errorf("failed to update subject: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("subject not found")
	}
	s.UpdatedAt = now
	return nil
}

// Delete deletes a subject
func (sr *SubjectRepository) Delete(ctx context.Context, id int64) error {
	res, err := sr.db.ExecContext(ctx, `DELETE FROM subjects WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete subject: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("subject not found")
	}
	return nil
}

// List retrieves subjects ordered by name
func (sr *SubjectRepository) List(ctx context.Context, limit, offset int) ([]*models.Subject, error) {
	rows, err := sr.db.QueryContext(ctx,
		`SELECT `+subjectColumns+` FROM subjects ORDER BY name LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list subjects: %w", err)
	}
	defer rows.Close()

	subjects := []*models.Subject{}
	for rows.Next() {
		s, err := scanSubject(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan subject: %w", err)
		}
		subjects = append(subjects, s)
	}
	return subjects, rows.Err()
}

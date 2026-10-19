package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/mzviagintsev/school-management/internal/models"
)

// GradeRepository handles grade database operations
type GradeRepository struct {
	db *sql.DB
}

// NewGradeRepository creates a new grade repository
func NewGradeRepository(db *sql.DB) *GradeRepository {
	return &GradeRepository{db: db}
}

// Create inserts a grade
func (gr *GradeRepository) Create(ctx context.Context, g *models.Grade) error {
	query := `
		INSERT INTO grades (student_id, subject_id, class_id, marks, term, recorded_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		RETURNING id, created_at, updated_at
	`
	if err := gr.db.QueryRowContext(ctx, query, g.StudentID, g.SubjectID, g.ClassID, g.Marks, g.Term, g.RecordedBy, time.Now()).
		Scan(&g.ID, &g.CreatedAt, &g.UpdatedAt); err != nil {
		return fmt.Errorf("failed to create grade: %w", err)
	}
	return nil
}

// GetByID retrieves a grade (nil, nil if not found)
func (gr *GradeRepository) GetByID(ctx context.Context, id int64) (*models.Grade, error) {
	g := &models.Grade{}
	err := gr.db.QueryRowContext(ctx, `
		SELECT id, student_id, subject_id, class_id, marks, COALESCE(term, ''), recorded_by, created_at, updated_at
		FROM grades WHERE id = $1`, id).
		Scan(&g.ID, &g.StudentID, &g.SubjectID, &g.ClassID, &g.Marks, &g.Term, &g.RecordedBy, &g.CreatedAt, &g.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get grade: %w", err)
	}
	return g, nil
}

// Update updates a grade's marks and term
func (gr *GradeRepository) Update(ctx context.Context, g *models.Grade) error {
	now := time.Now()
	res, err := gr.db.ExecContext(ctx,
		`UPDATE grades SET marks = $1, term = $2, updated_at = $3 WHERE id = $4`, g.Marks, g.Term, now, g.ID)
	if err != nil {
		return fmt.Errorf("failed to update grade: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("grade not found")
	}
	g.UpdatedAt = now
	return nil
}

// Delete deletes a grade
func (gr *GradeRepository) Delete(ctx context.Context, id int64) error {
	res, err := gr.db.ExecContext(ctx, `DELETE FROM grades WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete grade: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("grade not found")
	}
	return nil
}

const gradeDetailsSelect = `
	SELECT g.id, g.student_id, g.subject_id, g.class_id, g.marks, COALESCE(g.term, ''), g.recorded_by,
	       g.created_at, g.updated_at,
	       sub.name, sub.code, sub.credits, u.first_name, u.last_name
	FROM grades g
	JOIN subjects sub ON sub.id = g.subject_id
	JOIN students s ON s.id = g.student_id
	JOIN users u ON u.id = s.user_id
`

func (gr *GradeRepository) queryDetails(ctx context.Context, query string, args ...any) ([]*models.GradeWithDetails, error) {
	rows, err := gr.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list grades: %w", err)
	}
	defer rows.Close()

	result := []*models.GradeWithDetails{}
	for rows.Next() {
		g := &models.GradeWithDetails{}
		if err := rows.Scan(&g.ID, &g.StudentID, &g.SubjectID, &g.ClassID, &g.Marks, &g.Term, &g.RecordedBy,
			&g.CreatedAt, &g.UpdatedAt, &g.SubjectName, &g.SubjectCode, &g.Credits, &g.FirstName, &g.LastName); err != nil {
			return nil, fmt.Errorf("failed to scan grade: %w", err)
		}
		result = append(result, g)
	}
	return result, rows.Err()
}

// ListByStudent returns a student's grades (all terms when term is empty)
func (gr *GradeRepository) ListByStudent(ctx context.Context, studentID int64, term string) ([]*models.GradeWithDetails, error) {
	return gr.queryDetails(ctx, gradeDetailsSelect+`
		WHERE g.student_id = $1 AND ($2 = '' OR g.term = $2)
		ORDER BY sub.name, g.created_at`, studentID, term)
}

// ListByClassAndSubject returns a class's grades in one subject (all terms when term is empty)
func (gr *GradeRepository) ListByClassAndSubject(ctx context.Context, classID, subjectID int64, term string) ([]*models.GradeWithDetails, error) {
	return gr.queryDetails(ctx, gradeDetailsSelect+`
		WHERE g.class_id = $1 AND g.subject_id = $2 AND ($3 = '' OR g.term = $3)
		ORDER BY u.last_name, u.first_name, g.created_at`, classID, subjectID, term)
}

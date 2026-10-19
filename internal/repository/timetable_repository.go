package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/mzviagintsev/school-management/internal/models"
)

// TimetableRepository handles timetable database operations
type TimetableRepository struct {
	db *sql.DB
}

// NewTimetableRepository creates a new timetable repository
func NewTimetableRepository(db *sql.DB) *TimetableRepository {
	return &TimetableRepository{db: db}
}

// Times are stored as PostgreSQL TIME and always returned as "HH:MM".
const timetableColumns = `t.id, t.class_id, t.subject_id, t.teacher_id, t.day_of_week,
	to_char(t.start_time, 'HH24:MI'), to_char(t.end_time, 'HH24:MI'), COALESCE(t.room_number, ''),
	t.created_at, t.updated_at`

func scanTimetable(row interface{ Scan(...any) error }, extra ...any) (*models.Timetable, error) {
	t := &models.Timetable{}
	dest := []any{&t.ID, &t.ClassID, &t.SubjectID, &t.TeacherID, &t.DayOfWeek,
		&t.StartTime, &t.EndTime, &t.RoomNumber, &t.CreatedAt, &t.UpdatedAt}
	err := row.Scan(append(dest, extra...)...)
	return t, err
}

// Create inserts a timetable entry
func (tr *TimetableRepository) Create(ctx context.Context, e *models.Timetable) error {
	query := `
		INSERT INTO timetable (class_id, subject_id, teacher_id, day_of_week, start_time, end_time, room_number, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5::time, $6::time, $7, $8, $8)
		RETURNING id, created_at, updated_at
	`
	if err := tr.db.QueryRowContext(ctx, query, e.ClassID, e.SubjectID, e.TeacherID, e.DayOfWeek,
		e.StartTime, e.EndTime, e.RoomNumber, time.Now()).Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt); err != nil {
		return fmt.Errorf("failed to create timetable entry: %w", err)
	}
	return nil
}

// GetByID retrieves a timetable entry (nil, nil if not found)
func (tr *TimetableRepository) GetByID(ctx context.Context, id int64) (*models.Timetable, error) {
	t, err := scanTimetable(tr.db.QueryRowContext(ctx, `SELECT `+timetableColumns+` FROM timetable t WHERE t.id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get timetable entry: %w", err)
	}
	return t, nil
}

// Update replaces a timetable entry
func (tr *TimetableRepository) Update(ctx context.Context, e *models.Timetable) error {
	now := time.Now()
	res, err := tr.db.ExecContext(ctx, `
		UPDATE timetable
		SET class_id = $1, subject_id = $2, teacher_id = $3, day_of_week = $4,
		    start_time = $5::time, end_time = $6::time, room_number = $7, updated_at = $8
		WHERE id = $9`,
		e.ClassID, e.SubjectID, e.TeacherID, e.DayOfWeek, e.StartTime, e.EndTime, e.RoomNumber, now, e.ID)
	if err != nil {
		return fmt.Errorf("failed to update timetable entry: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("timetable entry not found")
	}
	e.UpdatedAt = now
	return nil
}

// Delete deletes a timetable entry
func (tr *TimetableRepository) Delete(ctx context.Context, id int64) error {
	res, err := tr.db.ExecContext(ctx, `DELETE FROM timetable WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete timetable entry: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("timetable entry not found")
	}
	return nil
}

// ListByDay returns all entries on a day of the week
func (tr *TimetableRepository) ListByDay(ctx context.Context, day int) ([]*models.Timetable, error) {
	rows, err := tr.db.QueryContext(ctx, `SELECT `+timetableColumns+` FROM timetable t WHERE t.day_of_week = $1`, day)
	if err != nil {
		return nil, fmt.Errorf("failed to list timetable: %w", err)
	}
	defer rows.Close()

	result := []*models.Timetable{}
	for rows.Next() {
		t, err := scanTimetable(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan timetable entry: %w", err)
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

func (tr *TimetableRepository) listDetails(ctx context.Context, where string, arg any) ([]*models.TimetableWithDetails, error) {
	rows, err := tr.db.QueryContext(ctx, `
		SELECT `+timetableColumns+`, c.name, sub.name, u.first_name || ' ' || u.last_name
		FROM timetable t
		JOIN classes c ON c.id = t.class_id
		JOIN subjects sub ON sub.id = t.subject_id
		JOIN teachers te ON te.id = t.teacher_id
		JOIN users u ON u.id = te.user_id
		WHERE `+where+`
		ORDER BY t.day_of_week, t.start_time`, arg)
	if err != nil {
		return nil, fmt.Errorf("failed to list timetable: %w", err)
	}
	defer rows.Close()

	result := []*models.TimetableWithDetails{}
	for rows.Next() {
		d := &models.TimetableWithDetails{}
		t, err := scanTimetable(rows, &d.ClassName, &d.SubjectName, &d.TeacherName)
		if err != nil {
			return nil, fmt.Errorf("failed to scan timetable entry: %w", err)
		}
		d.Timetable = *t
		result = append(result, d)
	}
	return result, rows.Err()
}

// ListByClass returns a class's weekly timetable ordered by day and start time
func (tr *TimetableRepository) ListByClass(ctx context.Context, classID int64) ([]*models.TimetableWithDetails, error) {
	return tr.listDetails(ctx, "t.class_id = $1", classID)
}

// ListByTeacher returns a teacher's weekly timetable ordered by day and start time
func (tr *TimetableRepository) ListByTeacher(ctx context.Context, teacherID int64) ([]*models.TimetableWithDetails, error) {
	return tr.listDetails(ctx, "t.teacher_id = $1", teacherID)
}

// TeacherTeachesClass reports whether a teacher has at least one lesson with a class
func (tr *TimetableRepository) TeacherTeachesClass(ctx context.Context, teacherID, classID int64) (bool, error) {
	var exists bool
	err := tr.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM timetable WHERE teacher_id = $1 AND class_id = $2)`, teacherID, classID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check timetable: %w", err)
	}
	return exists, nil
}

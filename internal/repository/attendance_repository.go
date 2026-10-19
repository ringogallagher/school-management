package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/mzviagintsev/school-management/internal/models"
)

// AttendanceRepository handles attendance database operations
type AttendanceRepository struct {
	db *sql.DB
}

// NewAttendanceRepository creates a new attendance repository
func NewAttendanceRepository(db *sql.DB) *AttendanceRepository {
	return &AttendanceRepository{db: db}
}

// UpsertBatch inserts all records in one transaction. A record that already exists
// for the same (student, class, date) is overwritten, so a teacher can correct a sheet.
func (ar *AttendanceRepository) UpsertBatch(ctx context.Context, records []*models.Attendance) error {
	tx, err := ar.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // no-op after Commit

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO attendance (student_id, class_id, date, status, remarks, recorded_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		ON CONFLICT (student_id, class_id, date) DO UPDATE
		SET status = EXCLUDED.status,
		    remarks = EXCLUDED.remarks,
		    recorded_by = EXCLUDED.recorded_by,
		    updated_at = EXCLUDED.updated_at
		RETURNING id, created_at, updated_at
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare attendance upsert: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, r := range records {
		if err := stmt.QueryRowContext(ctx, r.StudentID, r.ClassID, r.Date, r.Status, r.Remarks, r.RecordedBy, now).
			Scan(&r.ID, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return fmt.Errorf("failed to save attendance for student %d: %w", r.StudentID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit attendance: %w", err)
	}
	return nil
}

// ListByClassAndDate returns a class's attendance sheet for one day, with student names
func (ar *AttendanceRepository) ListByClassAndDate(ctx context.Context, classID int64, date time.Time) ([]*models.AttendanceWithStudent, error) {
	rows, err := ar.db.QueryContext(ctx, `
		SELECT a.id, a.student_id, a.class_id, a.date, a.status, COALESCE(a.remarks, ''), a.recorded_by,
		       a.created_at, a.updated_at, s.student_number, u.first_name, u.last_name
		FROM attendance a
		JOIN students s ON s.id = a.student_id
		JOIN users u ON u.id = s.user_id
		WHERE a.class_id = $1 AND a.date = $2
		ORDER BY u.last_name, u.first_name`, classID, date)
	if err != nil {
		return nil, fmt.Errorf("failed to list attendance: %w", err)
	}
	defer rows.Close()

	result := []*models.AttendanceWithStudent{}
	for rows.Next() {
		a := &models.AttendanceWithStudent{}
		if err := rows.Scan(&a.ID, &a.StudentID, &a.ClassID, &a.Date, &a.Status, &a.Remarks, &a.RecordedBy,
			&a.CreatedAt, &a.UpdatedAt, &a.StudentNumber, &a.FirstName, &a.LastName); err != nil {
			return nil, fmt.Errorf("failed to scan attendance: %w", err)
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

// ListByStudent returns a student's attendance between from and to (inclusive), newest first
func (ar *AttendanceRepository) ListByStudent(ctx context.Context, studentID int64, from, to time.Time) ([]*models.Attendance, error) {
	rows, err := ar.db.QueryContext(ctx, `
		SELECT id, student_id, class_id, date, status, COALESCE(remarks, ''), recorded_by, created_at, updated_at
		FROM attendance
		WHERE student_id = $1 AND date BETWEEN $2 AND $3
		ORDER BY date DESC`, studentID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to list attendance: %w", err)
	}
	defer rows.Close()

	result := []*models.Attendance{}
	for rows.Next() {
		a := &models.Attendance{}
		if err := rows.Scan(&a.ID, &a.StudentID, &a.ClassID, &a.Date, &a.Status, &a.Remarks, &a.RecordedBy,
			&a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan attendance: %w", err)
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

// SummaryByClass counts attendance statuses per student of a class in a date range.
// Students with no records in the range are included with zero counts.
// The percentage is filled in by the service layer.
func (ar *AttendanceRepository) SummaryByClass(ctx context.Context, classID int64, from, to time.Time) ([]*models.AttendanceSummary, error) {
	rows, err := ar.db.QueryContext(ctx, `
		SELECT s.id, u.first_name, u.last_name,
		       COUNT(a.id),
		       COUNT(a.id) FILTER (WHERE a.status = 'present'),
		       COUNT(a.id) FILTER (WHERE a.status = 'absent'),
		       COUNT(a.id) FILTER (WHERE a.status = 'late'),
		       COUNT(a.id) FILTER (WHERE a.status = 'excused')
		FROM students s
		JOIN users u ON u.id = s.user_id
		LEFT JOIN attendance a
		       ON a.student_id = s.id AND a.class_id = s.class_id AND a.date BETWEEN $2 AND $3
		WHERE s.class_id = $1 AND u.deleted_at IS NULL
		GROUP BY s.id, u.first_name, u.last_name
		ORDER BY u.last_name, u.first_name`, classID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to summarise attendance: %w", err)
	}
	defer rows.Close()

	result := []*models.AttendanceSummary{}
	for rows.Next() {
		s := &models.AttendanceSummary{}
		if err := rows.Scan(&s.StudentID, &s.FirstName, &s.LastName,
			&s.Total, &s.Present, &s.Absent, &s.Late, &s.Excused); err != nil {
			return nil, fmt.Errorf("failed to scan attendance summary: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// ClassesMissingAttendance finds classes that have at least one timetable lesson on the
// given weekday but no attendance record for the given date, with their homeroom teacher.
func (ar *AttendanceRepository) ClassesMissingAttendance(ctx context.Context, dayOfWeek int, date time.Time) ([]*models.ReminderTarget, error) {
	rows, err := ar.db.QueryContext(ctx, `
		SELECT c.id, c.name, c.teacher_id
		FROM classes c
		WHERE EXISTS (SELECT 1 FROM timetable t WHERE t.class_id = c.id AND t.day_of_week = $1)
		  AND NOT EXISTS (SELECT 1 FROM attendance a WHERE a.class_id = c.id AND a.date = $2)
		ORDER BY c.name`, dayOfWeek, date)
	if err != nil {
		return nil, fmt.Errorf("failed to find classes missing attendance: %w", err)
	}
	defer rows.Close()

	result := []*models.ReminderTarget{}
	for rows.Next() {
		t := &models.ReminderTarget{}
		if err := rows.Scan(&t.ClassID, &t.ClassName, &t.TeacherUserID); err != nil {
			return nil, fmt.Errorf("failed to scan reminder target: %w", err)
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

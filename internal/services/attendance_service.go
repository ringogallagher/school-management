package services

import (
	"context"
	"fmt"
	"time"

	"github.com/mzviagintsev/school-management/internal/models"
)

const (
	dateLayout = "2006-01-02"

	// LowAttendanceThreshold is the percentage below which students and parents are warned.
	LowAttendanceThreshold = 75.0
	// lowAttendanceWindow is the look-back period used for the low-attendance check.
	lowAttendanceWindow = 30 * 24 * time.Hour
	// minDaysForLowAttendance avoids warning after only one or two recorded days.
	minDaysForLowAttendance = 5
)

// AttendanceService handles attendance business logic
type AttendanceService struct {
	attendanceRepo AttendanceRepository
	studentRepo    StudentRepository
	classRepo      ClassRepository
	access         *AccessControl
	notifier       *NotificationService
	now            func() time.Time // injectable clock for tests
}

// NewAttendanceService creates a new attendance service
func NewAttendanceService(attendanceRepo AttendanceRepository, studentRepo StudentRepository, classRepo ClassRepository, access *AccessControl, notifier *NotificationService) *AttendanceService {
	return &AttendanceService{
		attendanceRepo: attendanceRepo,
		studentRepo:    studentRepo,
		classRepo:      classRepo,
		access:         access,
		notifier:       notifier,
		now:            time.Now,
	}
}

func isValidAttendanceStatus(s models.AttendanceStatus) bool {
	switch s {
	case models.AttendancePresent, models.AttendanceAbsent, models.AttendanceLate, models.AttendanceExcused:
		return true
	}
	return false
}

// truncateToDate drops the time-of-day part, keeping the date in UTC
func truncateToDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// parseDateOr parses YYYY-MM-DD, or returns def when the string is empty
func parseDateOr(s string, def time.Time) (time.Time, error) {
	if s == "" {
		return truncateToDate(def), nil
	}
	d, err := time.Parse(dateLayout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q, expected YYYY-MM-DD", s)
	}
	return d, nil
}

// MarkAttendance records attendance for several students of one class on one day.
// The whole batch is validated first and written atomically; re-marking the same
// day overwrites earlier records. Absences trigger notifications to student and
// parents, and so does attendance dropping below LowAttendanceThreshold.
func (as *AttendanceService) MarkAttendance(ctx context.Context, actor Actor, req *models.MarkAttendanceRequest) ([]*models.Attendance, error) {
	if req.ClassID == 0 {
		return nil, fmt.Errorf("class_id is required")
	}
	if len(req.Records) == 0 {
		return nil, fmt.Errorf("records must not be empty")
	}

	today := truncateToDate(as.now())
	date, err := parseDateOr(req.Date, today)
	if err != nil {
		return nil, err
	}
	if date.After(today) {
		return nil, fmt.Errorf("cannot mark attendance for a future date")
	}

	if err := as.access.CanTeachClass(ctx, actor, req.ClassID); err != nil {
		return nil, err
	}

	seen := make(map[int64]bool, len(req.Records))
	students := make(map[int64]*models.Student, len(req.Records))
	records := make([]*models.Attendance, 0, len(req.Records))
	recordedBy := actor.UserID

	for _, entry := range req.Records {
		if seen[entry.StudentID] {
			return nil, fmt.Errorf("student %d appears more than once", entry.StudentID)
		}
		seen[entry.StudentID] = true

		status := entry.Status
		if status == "" {
			status = models.AttendancePresent
		}
		if !isValidAttendanceStatus(status) {
			return nil, fmt.Errorf("invalid status %q for student %d", entry.Status, entry.StudentID)
		}

		student, err := as.studentRepo.GetByID(ctx, entry.StudentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get student %d: %w", entry.StudentID, err)
		}
		if student == nil {
			return nil, fmt.Errorf("student %d: %w", entry.StudentID, ErrNotFound)
		}
		if student.ClassID != req.ClassID {
			return nil, fmt.Errorf("student %d is not enrolled in class %d", entry.StudentID, req.ClassID)
		}
		students[student.ID] = student

		records = append(records, &models.Attendance{
			StudentID:  entry.StudentID,
			ClassID:    req.ClassID,
			Date:       date,
			Status:     status,
			Remarks:    entry.Remarks,
			RecordedBy: &recordedBy,
		})
	}

	if err := as.attendanceRepo.UpsertBatch(ctx, records); err != nil {
		return nil, fmt.Errorf("failed to save attendance: %w", err)
	}

	for _, rec := range records {
		student := students[rec.StudentID]
		if rec.Status == models.AttendanceAbsent {
			as.notifier.NotifyStudentAndParents(ctx, student, models.NotificationAbsence,
				"Absence recorded",
				fmt.Sprintf("An absence was recorded on %s.", date.Format(dateLayout)))
		}
		// Any record can be the one that pushes the student below the threshold
		// (e.g. a late or present mark that brings the counted days up to the minimum).
		as.checkLowAttendance(ctx, student, date)
	}

	return records, nil
}

// checkLowAttendance warns student and parents when attendance over the last 30 days
// (counted back from today, so back-filled days are included) drops below the threshold.
// It only fires when the percentage *crosses* the threshold because of the record just
// saved for markedDate, so a student who is already below it is not warned on every absence.
func (as *AttendanceService) checkLowAttendance(ctx context.Context, student *models.Student, markedDate time.Time) {
	today := truncateToDate(as.now())
	records, err := as.attendanceRepo.ListByStudent(ctx, student.ID, today.Add(-lowAttendanceWindow), today)
	if err != nil {
		return
	}

	now := SummarizeAttendance(student.ID, records)
	if now.Total < minDaysForLowAttendance || now.Percentage >= LowAttendanceThreshold {
		return
	}

	previous := make([]*models.Attendance, 0, len(records))
	for _, r := range records {
		if !r.Date.Equal(markedDate) {
			previous = append(previous, r)
		}
	}
	before := SummarizeAttendance(student.ID, previous)
	alreadyBelow := before.Total >= minDaysForLowAttendance && before.Percentage < LowAttendanceThreshold
	if alreadyBelow {
		return
	}

	as.notifier.NotifyStudentAndParents(ctx, student, models.NotificationAttendanceLow,
		"Low attendance warning",
		fmt.Sprintf("Attendance over the last 30 days is %.1f%%, below the required %.0f%%.", now.Percentage, LowAttendanceThreshold))
}

// SummarizeAttendance counts statuses and computes the attendance percentage.
// Late counts as attended; excused days are excluded from the denominator.
func SummarizeAttendance(studentID int64, records []*models.Attendance) models.AttendanceSummary {
	s := models.AttendanceSummary{StudentID: studentID}
	for _, r := range records {
		s.Total++
		switch r.Status {
		case models.AttendancePresent:
			s.Present++
		case models.AttendanceAbsent:
			s.Absent++
		case models.AttendanceLate:
			s.Late++
		case models.AttendanceExcused:
			s.Excused++
		}
	}
	s.Percentage = attendancePercentage(s.Present, s.Late, s.Total, s.Excused)
	return s
}

func attendancePercentage(present, late, total, excused int) float64 {
	counted := total - excused
	if counted <= 0 {
		return 100
	}
	return roundTo2(float64(present+late) / float64(counted) * 100)
}

// GetClassAttendance returns the attendance sheet of a class for one day
func (as *AttendanceService) GetClassAttendance(ctx context.Context, actor Actor, classID int64, dateStr string) ([]*models.AttendanceWithStudent, error) {
	date, err := parseDateOr(dateStr, as.now())
	if err != nil {
		return nil, err
	}
	if err := as.access.CanTeachClass(ctx, actor, classID); err != nil {
		return nil, err
	}
	records, err := as.attendanceRepo.ListByClassAndDate(ctx, classID, date)
	if err != nil {
		return nil, fmt.Errorf("failed to get class attendance: %w", err)
	}
	return records, nil
}

// resolveRange parses from/to (defaults: last 30 days up to today) and validates order
func (as *AttendanceService) resolveRange(fromStr, toStr string) (time.Time, time.Time, error) {
	to, err := parseDateOr(toStr, as.now())
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	from, err := parseDateOr(fromStr, to.Add(-lowAttendanceWindow))
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if from.After(to) {
		return time.Time{}, time.Time{}, fmt.Errorf("from must not be after to")
	}
	return from, to, nil
}

// GetStudentReport returns a student's attendance records and summary for a date range
func (as *AttendanceService) GetStudentReport(ctx context.Context, actor Actor, studentID int64, fromStr, toStr string) (*models.StudentAttendanceReport, error) {
	from, to, err := as.resolveRange(fromStr, toStr)
	if err != nil {
		return nil, err
	}
	student, err := as.studentRepo.GetByID(ctx, studentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get student: %w", err)
	}
	if student == nil {
		return nil, fmt.Errorf("student: %w", ErrNotFound)
	}
	if err := as.access.CanViewStudent(ctx, actor, student); err != nil {
		return nil, err
	}

	records, err := as.attendanceRepo.ListByStudent(ctx, studentID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get attendance: %w", err)
	}
	if records == nil {
		records = []*models.Attendance{}
	}
	return &models.StudentAttendanceReport{
		Summary: SummarizeAttendance(studentID, records),
		Records: records,
	}, nil
}

// GetMyReport returns the calling student's own attendance report
func (as *AttendanceService) GetMyReport(ctx context.Context, actor Actor, fromStr, toStr string) (*models.StudentAttendanceReport, error) {
	student, err := as.studentRepo.GetByUserID(ctx, actor.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get student profile: %w", err)
	}
	if student == nil {
		return nil, fmt.Errorf("student profile: %w", ErrNotFound)
	}
	return as.GetStudentReport(ctx, actor, student.ID, fromStr, toStr)
}

// GetClassStatistics returns per-student attendance summaries for a class over a date range
func (as *AttendanceService) GetClassStatistics(ctx context.Context, actor Actor, classID int64, fromStr, toStr string) ([]*models.AttendanceSummary, error) {
	from, to, err := as.resolveRange(fromStr, toStr)
	if err != nil {
		return nil, err
	}
	if err := as.access.CanTeachClass(ctx, actor, classID); err != nil {
		return nil, err
	}
	stats, err := as.attendanceRepo.SummaryByClass(ctx, classID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get class statistics: %w", err)
	}
	for _, s := range stats {
		s.Percentage = attendancePercentage(s.Present, s.Late, s.Total, s.Excused)
	}
	return stats, nil
}

package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mzviagintsev/school-management/internal/models"
)

// NotificationAttendanceReminder is sent to homeroom teachers who have not marked attendance yet
const NotificationAttendanceReminder = "attendance_reminder"

// ReminderService reminds homeroom teachers, once a day, to mark attendance for
// classes that had lessons today but have no attendance recorded.
type ReminderService struct {
	attendanceRepo AttendanceRepository
	notifier       *NotificationService
	now            func() time.Time
}

// NewReminderService creates the reminder job
func NewReminderService(attendanceRepo AttendanceRepository, notifier *NotificationService) *ReminderService {
	return &ReminderService{attendanceRepo: attendanceRepo, notifier: notifier, now: time.Now}
}

// RunOnce sends reminders for today and returns how many were sent
func (rs *ReminderService) RunOnce(ctx context.Context) (int, error) {
	now := rs.now()
	today := truncateToDate(now)
	targets, err := rs.attendanceRepo.ClassesMissingAttendance(ctx, int(now.Weekday()), today)
	if err != nil {
		return 0, err
	}
	for _, t := range targets {
		rs.notifier.NotifyUsers(ctx, []int64{t.TeacherUserID}, NotificationAttendanceReminder,
			"Attendance not marked",
			fmt.Sprintf("Attendance for class %s has not been recorded for %s yet.", t.ClassName, today.Format(dateLayout)))
	}
	return len(targets), nil
}

// nextRun returns the next occurrence of hour:00 strictly after now
func nextRun(now time.Time, hour int) time.Time {
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

// Start runs RunOnce every day at the given hour (server local time) until ctx is cancelled
func (rs *ReminderService) Start(ctx context.Context, hour int) {
	go func() {
		for {
			wait := time.Until(nextRun(rs.now(), hour))
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				n, err := rs.RunOnce(ctx)
				if err != nil {
					slog.Warn("attendance reminder job failed", "error", err)
				} else {
					slog.Info("attendance reminders sent", "count", n)
				}
			}
		}
	}()
}

// Pending returns today's classes that have lessons but no attendance yet (for dashboards)
func (rs *ReminderService) Pending(ctx context.Context) ([]*models.ReminderTarget, error) {
	now := rs.now()
	return rs.attendanceRepo.ClassesMissingAttendance(ctx, int(now.Weekday()), truncateToDate(now))
}

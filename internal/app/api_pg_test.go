//go:build integration

package app

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/mzviagintsev/school-management/internal/db"
	appMiddleware "github.com/mzviagintsev/school-management/internal/middleware"
	"github.com/mzviagintsev/school-management/internal/repository"
	"github.com/mzviagintsev/school-management/internal/services"
	"net/http/httptest"
)

// TestAPI_PostgreSQL runs the same scenarios as the in-memory test against a real database.
//
//	make migrate-up            # or apply migrations/*.up.sql
//	TEST_DATABASE_URL=postgres://... go test -tags integration ./internal/app/
func TestAPI_PostgreSQL(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	database, err := db.New(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	c := database.Conn()

	userRepo := repository.NewUserRepository(c)
	classRepo := repository.NewClassRepository(c)
	teacherRepo := repository.NewTeacherRepository(c)
	studentRepo := repository.NewStudentRepository(c)
	subjectRepo := repository.NewSubjectRepository(c)
	parentRepo := repository.NewParentRepository(c)
	attendanceRepo := repository.NewAttendanceRepository(c)
	gradeRepo := repository.NewGradeRepository(c)
	timetableRepo := repository.NewTimetableRepository(c)
	notificationRepo := repository.NewNotificationRepository(c)
	newsRepo := repository.NewNewsRepository(c)

	access := services.NewAccessControl(classRepo, teacherRepo, parentRepo, timetableRepo)
	notifier := services.NewNotificationService(notificationRepo, parentRepo)
	d := Deps{
		Auth:          services.NewAuthService(userRepo, "test-secret"),
		Users:         services.NewUserService(userRepo),
		Classes:       services.NewClassService(classRepo, userRepo),
		Teachers:      services.NewTeacherService(teacherRepo, userRepo),
		Students:      services.NewStudentService(studentRepo, classRepo, userRepo),
		Subjects:      services.NewSubjectService(subjectRepo, teacherRepo),
		Parents:       services.NewParentService(parentRepo, studentRepo, userRepo),
		Attendance:    services.NewAttendanceService(attendanceRepo, studentRepo, classRepo, access, notifier),
		Grades:        services.NewGradeService(gradeRepo, studentRepo, subjectRepo, access, notifier),
		Timetable:     services.NewTimetableService(timetableRepo, classRepo, subjectRepo, teacherRepo, studentRepo),
		Notifications: notifier,
		Reminders:     services.NewReminderService(attendanceRepo, notifier),
		News:          services.NewNewsService(newsRepo),
		Access:        access,
		LoginLimiter:  appMiddleware.NewRateLimiter(1000, time.Minute),
	}
	const adminEmail, adminPass = "it-admin@school.test", "it-admin-pass-1"
	if err := d.Users.EnsureAdmin(context.Background(), adminEmail, adminPass); err != nil {
		t.Fatal(err)
	}
	h, err := NewRouter(d)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()
	runScenarios(t, srv.URL, adminEmail, adminPass)
}

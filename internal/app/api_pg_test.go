//go:build integration

package app

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/mzviagintsev/school-management/internal/db"
	appMiddleware "github.com/mzviagintsev/school-management/internal/middleware"
	"github.com/mzviagintsev/school-management/internal/models"
	"github.com/mzviagintsev/school-management/internal/repository"
	"github.com/mzviagintsev/school-management/internal/services"
	"net/http/httptest"
)

// TestAPI_PostgreSQL runs the same scenarios as the in-memory test against a real database.
// The database must be migrated; it may already contain data (the test creates its own administrator).
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
	// A dedicated administrator with a unique e-mail, created directly in the repository, so the
	// test also works on a database that already has an administrator (for example after cmd/seed).
	adminEmail := fmt.Sprintf("it-admin-%d@school.test", time.Now().UnixNano())
	const adminPass = "it-admin-pass-1"
	hash, err := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := userRepo.Create(context.Background(), &models.User{
		Email: adminEmail, PasswordHash: string(hash), FirstName: "IT", LastName: "Admin",
		Role: models.RoleAdmin, IsActive: true,
	}); err != nil {
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

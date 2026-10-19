package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mzviagintsev/school-management/internal/memrepo"
	appMiddleware "github.com/mzviagintsev/school-management/internal/middleware"
	"github.com/mzviagintsev/school-management/internal/services"
)

const (
	testAdminEmail    = "admin@school.test"
	testAdminPassword = "admin12345"
)

// memoryDeps wires every service to in-memory repositories
func memoryDeps(limiter *appMiddleware.RateLimiter) Deps {
	users := memrepo.NewMockUserRepository()
	classes := memrepo.NewMockClassRepository()
	teachers := memrepo.NewMockTeacherRepository()
	students := memrepo.NewMockStudentRepository()
	subjects := memrepo.NewMockSubjectRepository()
	parents := memrepo.NewMockParentRepository()
	attendance := memrepo.NewMockAttendanceRepository()
	grades := memrepo.NewMockGradeRepository(subjects)
	timetable := memrepo.NewMockTimetableRepository()
	notifications := memrepo.NewMockNotificationRepository()
	news := memrepo.NewMockNewsRepository()

	access := services.NewAccessControl(classes, teachers, parents, timetable)
	notifier := services.NewNotificationService(notifications, parents)
	return Deps{
		Auth:          services.NewAuthService(users, "test-secret"),
		Users:         services.NewUserService(users),
		Classes:       services.NewClassService(classes, users),
		Teachers:      services.NewTeacherService(teachers, users),
		Students:      services.NewStudentService(students, classes, users),
		Subjects:      services.NewSubjectService(subjects, teachers),
		Parents:       services.NewParentService(parents, students, users),
		Attendance:    services.NewAttendanceService(attendance, students, classes, access, notifier),
		Grades:        services.NewGradeService(grades, students, subjects, access, notifier),
		Timetable:     services.NewTimetableService(timetable, classes, subjects, teachers, students),
		Notifications: notifier,
		Reminders:     services.NewReminderService(attendance, notifier),
		News:          services.NewNewsService(news),
		Access:        access,
		LoginLimiter:  limiter,
	}
}

func newMemoryServer(t *testing.T, limiter *appMiddleware.RateLimiter) *httptest.Server {
	t.Helper()
	d := memoryDeps(limiter)
	if err := d.Users.EnsureAdmin(context.Background(), testAdminEmail, testAdminPassword); err != nil {
		t.Fatal(err)
	}
	h, err := NewRouter(d)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

func TestAPI_InMemory(t *testing.T) {
	srv := newMemoryServer(t, appMiddleware.NewRateLimiter(1000, time.Minute))
	runScenarios(t, srv.URL, testAdminEmail, testAdminPassword)
}

// Ten wrong passwords in a minute from one address lock the login out (API and web form share the limiter)
func TestLoginRateLimit(t *testing.T) {
	srv := newMemoryServer(t, appMiddleware.NewRateLimiter(5, time.Minute))
	c := newClient(t, srv.URL)
	var last reply
	for i := 0; i < 6; i++ {
		last = c.post("/api/auth/login", "", map[string]string{"email": testAdminEmail, "password": "wrong-wrong"})
		if i < 5 && last.Status != http.StatusUnauthorized {
			t.Fatalf("attempt %d: got %d, want 401", i+1, last.Status)
		}
	}
	if last.Status != http.StatusTooManyRequests {
		t.Errorf("6th attempt: got %d, want 429", last.Status)
	}
	if last.Header.Get("Retry-After") == "" {
		t.Error("429 should carry Retry-After")
	}
	// even the right password is refused while locked out
	if r := c.post("/api/auth/login", "", map[string]string{"email": testAdminEmail, "password": testAdminPassword}); r.Status != http.StatusTooManyRequests {
		t.Errorf("locked-out login: got %d, want 429", r.Status)
	}
}

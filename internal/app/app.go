// Package app assembles the HTTP router (JSON API under /api plus the HTMX frontend) from
// ready-made services. cmd/api uses it with PostgreSQL repositories; the integration tests
// use it with in-memory or real repositories, so tests exercise exactly the production routes,
// middleware and handlers.
package app

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/mzviagintsev/school-management/internal/handlers"
	appMiddleware "github.com/mzviagintsev/school-management/internal/middleware"
	"github.com/mzviagintsev/school-management/internal/models"
	"github.com/mzviagintsev/school-management/internal/services"
	"github.com/mzviagintsev/school-management/internal/web"
)

// Deps are the services the router needs
type Deps struct {
	Auth          *services.AuthService
	Users         *services.UserService
	Classes       *services.ClassService
	Teachers      *services.TeacherService
	Students      *services.StudentService
	Subjects      *services.SubjectService
	Parents       *services.ParentService
	Attendance    *services.AttendanceService
	Grades        *services.GradeService
	Timetable     *services.TimetableService
	Notifications *services.NotificationService
	Reminders     *services.ReminderService
	News          *services.NewsService
	Access        *services.AccessControl

	LoginLimiter *appMiddleware.RateLimiter
	SecureCookie bool // mark the session cookie Secure (HTTPS deployments)
	AccessLog    bool // log every request
}

// NewRouter builds the complete router
func NewRouter(d Deps) (http.Handler, error) {
	// Initialize handlers
	authHandler := handlers.NewAuthHandler(d.Auth)
	classHandler := handlers.NewClassHandler(d.Classes)
	teacherHandler := handlers.NewTeacherHandler(d.Teachers)
	studentHandler := handlers.NewStudentHandler(d.Students)
	subjectHandler := handlers.NewSubjectHandler(d.Subjects)
	parentHandler := handlers.NewParentHandler(d.Parents)
	attendanceHandler := handlers.NewAttendanceHandler(d.Attendance)
	gradeHandler := handlers.NewGradeHandler(d.Grades)
	timetableHandler := handlers.NewTimetableHandler(d.Timetable)
	notificationHandler := handlers.NewNotificationHandler(d.Notifications)
	userHandler := handlers.NewUserHandler(d.Users)
	newsHandler := handlers.NewNewsHandler(d.News)
	exportHandler := handlers.NewExportHandler(d.Grades, d.Attendance)

	webHandler, err := web.New(web.Services{
		Auth: d.Auth, Users: d.Users, Classes: d.Classes, Teachers: d.Teachers,
		Students: d.Students, Subjects: d.Subjects, Parents: d.Parents,
		Attendance: d.Attendance, Grades: d.Grades, Timetable: d.Timetable,
		Notifications: d.Notifications, Reminders: d.Reminders, News: d.News, Access: d.Access,
	}, d.SecureCookie, d.LoginLimiter)
	if err != nil {
		return nil, fmt.Errorf("load web templates: %w", err)
	}
	loginLimiter := d.LoginLimiter

	// Create router
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	if d.AccessLog {
		r.Use(middleware.Logger)
	}
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(appMiddleware.SecurityHeaders)

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok"}`)
	})

	// All API routes live under a single "/api" mount. Mounting "/api/auth" and "/api"
	// separately (as two distinct chi.Route trees) causes the second mount to shadow
	// routes registered by the first, so public and protected routes are kept together
	// here and split with r.Group instead.
	r.Route("/api", func(r chi.Router) {
		// Auth routes: login is public (rate limited); registering accounts is admin-only
		// (anyone could previously register themselves as an admin)
		r.Route("/auth", func(r chi.Router) {
			r.With(loginLimiter.Middleware).Post("/login", authHandler.Login)

			r.Group(func(r chi.Router) {
				r.Use(appMiddleware.AuthMiddleware(d.Auth))
				r.Get("/profile", authHandler.GetProfile)
				r.Post("/change-password", authHandler.ChangePassword)
				r.With(appMiddleware.RequireRole(models.RoleAdmin)).Post("/register", authHandler.Register)
			})
		})

		// School news is public to read (it is also shown on the public home page)
		r.Get("/news", newsHandler.List)
		r.Get("/news/{id}", newsHandler.GetByID)

		// Everything else under /api requires a valid token
		r.Group(func(r chi.Router) {
			r.Use(appMiddleware.AuthMiddleware(d.Auth))

			// Publishing news is admin-only
			r.With(appMiddleware.RequireRole(models.RoleAdmin)).Post("/news", newsHandler.Create)
			r.With(appMiddleware.RequireRole(models.RoleAdmin)).Put("/news/{id}", newsHandler.Update)
			r.With(appMiddleware.RequireRole(models.RoleAdmin)).Delete("/news/{id}", newsHandler.Delete)

			// User routes (example - add more as needed)
			r.Route("/users", func(r chi.Router) {
				r.Use(appMiddleware.RequireRole(models.RoleAdmin))
				r.Get("/", userHandler.List)
				r.Get("/{id}", userHandler.GetByID)
				r.Put("/{id}", userHandler.Update)
				r.Delete("/{id}", userHandler.Delete)
				r.Post("/{id}/reset-password", userHandler.ResetPassword)
			})

			// Class routes: any authenticated user can view; only admins can manage
			r.Route("/classes", func(r chi.Router) {
				r.Get("/", classHandler.List)
				r.Get("/{id}", classHandler.GetByID)

				r.Group(func(r chi.Router) {
					r.Use(appMiddleware.RequireRole(models.RoleAdmin))
					r.Post("/", classHandler.Create)
					r.Put("/{id}", classHandler.Update)
					r.Delete("/{id}", classHandler.Delete)
				})
			})

			// Teacher routes: admins and teachers can view; only admins can manage
			r.Route("/teachers", func(r chi.Router) {
				r.Group(func(r chi.Router) {
					r.Use(appMiddleware.RequireRole(models.RoleAdmin, models.RoleTeacher))
					r.Get("/", teacherHandler.List)
					r.Get("/{id}", teacherHandler.GetByID)
					r.Get("/{id}/subjects", subjectHandler.TeacherSubjects)
				})

				r.Group(func(r chi.Router) {
					r.Use(appMiddleware.RequireRole(models.RoleAdmin))
					r.Put("/{id}/subjects", subjectHandler.AssignToTeacher)
					r.Post("/", teacherHandler.Create)
					r.Put("/{id}", teacherHandler.Update)
					r.Delete("/{id}", teacherHandler.Delete)
				})
			})

			// Student routes: admins and teachers can view; only admins can manage
			r.Route("/students", func(r chi.Router) {
				r.Group(func(r chi.Router) {
					r.Use(appMiddleware.RequireRole(models.RoleAdmin, models.RoleTeacher))
					r.Get("/", studentHandler.List)
					r.Get("/{id}", studentHandler.GetByID)
					r.Get("/class/{classID}", studentHandler.ListByClass)
				})

				r.Group(func(r chi.Router) {
					r.Use(appMiddleware.RequireRole(models.RoleAdmin))
					r.Post("/", studentHandler.Create)
					r.Put("/{id}", studentHandler.Update)
					r.Delete("/{id}", studentHandler.Delete)
				})
			})

			// Subject routes: any authenticated user can view; only admins can manage
			r.Route("/subjects", func(r chi.Router) {
				r.Get("/", subjectHandler.List)
				r.Get("/{id}", subjectHandler.GetByID)

				r.Group(func(r chi.Router) {
					r.Use(appMiddleware.RequireRole(models.RoleAdmin))
					r.Post("/", subjectHandler.Create)
					r.Put("/{id}", subjectHandler.Update)
					r.Delete("/{id}", subjectHandler.Delete)
				})
			})

			// Parent routes: admins create parents and link them to students;
			// parents can list their own children
			r.Route("/parents", func(r chi.Router) {
				r.With(appMiddleware.RequireRole(models.RoleParent)).Get("/me/children", parentHandler.MyChildren)

				r.Group(func(r chi.Router) {
					r.Use(appMiddleware.RequireRole(models.RoleAdmin))
					r.Post("/", parentHandler.Create)
					r.Post("/{id}/students/{studentID}", parentHandler.LinkStudent)
					r.Delete("/{id}/students/{studentID}", parentHandler.UnlinkStudent)
				})
			})

			// Attendance routes. Fine-grained rules (which class a teacher may mark,
			// whose report a parent may see) are enforced in AccessControl.
			r.Route("/attendance", func(r chi.Router) {
				r.With(appMiddleware.RequireRole(models.RoleStudent)).Get("/me", attendanceHandler.MyReport)
				r.With(appMiddleware.RequireRole(models.RoleStudent)).Get("/me/export.csv", exportHandler.MyAttendance)
				r.Get("/student/{studentID}", attendanceHandler.StudentReport)
				r.Get("/student/{studentID}/export.csv", exportHandler.StudentAttendance)

				r.Group(func(r chi.Router) {
					r.Use(appMiddleware.RequireRole(models.RoleAdmin, models.RoleTeacher))
					r.Post("/", attendanceHandler.Mark)
					r.Get("/class/{classID}", attendanceHandler.ClassSheet)
					r.Get("/class/{classID}/stats", attendanceHandler.ClassStatistics)
					r.Get("/class/{classID}/stats.csv", exportHandler.ClassAttendance)
				})
			})

			// Grade routes
			r.Route("/grades", func(r chi.Router) {
				r.With(appMiddleware.RequireRole(models.RoleStudent)).Get("/me", gradeHandler.MyReport)
				r.With(appMiddleware.RequireRole(models.RoleStudent)).Get("/me/export.csv", exportHandler.MyGrades)
				r.Get("/student/{studentID}", gradeHandler.StudentReport)
				r.Get("/student/{studentID}/export.csv", exportHandler.StudentGrades)

				r.Group(func(r chi.Router) {
					r.Use(appMiddleware.RequireRole(models.RoleAdmin, models.RoleTeacher))
					r.Post("/", gradeHandler.Record)
					r.Put("/{id}", gradeHandler.Update)
					r.Delete("/{id}", gradeHandler.Delete)
					r.Get("/class/{classID}/subject/{subjectID}", gradeHandler.ClassSubject)
					r.Get("/class/{classID}/subject/{subjectID}/export.csv", exportHandler.ClassGrades)
				})
			})

			// Timetable routes: anyone authenticated can read; only admins can change
			r.Route("/timetable", func(r chi.Router) {
				r.With(appMiddleware.RequireRole(models.RoleTeacher, models.RoleStudent)).Get("/me", timetableHandler.Mine)
				r.Get("/class/{classID}", timetableHandler.ByClass)
				r.Get("/teacher/{teacherID}", timetableHandler.ByTeacher)

				r.Group(func(r chi.Router) {
					r.Use(appMiddleware.RequireRole(models.RoleAdmin))
					r.Post("/", timetableHandler.Create)
					r.Put("/{id}", timetableHandler.Update)
					r.Delete("/{id}", timetableHandler.Delete)
				})
			})

			// Notification routes: every user manages their own inbox; admins broadcast
			r.Route("/notifications", func(r chi.Router) {
				r.Get("/", notificationHandler.List)
				r.Get("/unread-count", notificationHandler.UnreadCount)
				r.Post("/read-all", notificationHandler.MarkAllRead)
				r.Post("/{id}/read", notificationHandler.MarkRead)
				r.With(appMiddleware.RequireRole(models.RoleAdmin)).Post("/announce", notificationHandler.Announce)
			})
		})
	})

	// HTML frontend (HTMX) at the site root
	webHandler.Routes(r)

	return r, nil
}

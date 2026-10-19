package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/mzviagintsev/school-management/internal/app"
	"github.com/mzviagintsev/school-management/internal/db"
	"github.com/mzviagintsev/school-management/internal/mailer"
	appMiddleware "github.com/mzviagintsev/school-management/internal/middleware"
	"github.com/mzviagintsev/school-management/internal/repository"
	"github.com/mzviagintsev/school-management/internal/services"
)

// envInt reads an integer environment variable with a default
func envInt(key string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil {
		return v
	}
	return def
}

func main() {
	// Initialize logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Load configuration from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:password@localhost:5432/school_management?sslmode=disable"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-secret-key-change-in-production" // Change in production!
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize database
	logger.Info("Connecting to database", "url", dbURL)
	database, err := db.New(dbURL)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	logger.Info("Successfully connected to database")

	// Initialize repositories
	userRepo := repository.NewUserRepository(database.Conn())
	classRepo := repository.NewClassRepository(database.Conn())
	teacherRepo := repository.NewTeacherRepository(database.Conn())
	studentRepo := repository.NewStudentRepository(database.Conn())
	subjectRepo := repository.NewSubjectRepository(database.Conn())
	parentRepo := repository.NewParentRepository(database.Conn())
	attendanceRepo := repository.NewAttendanceRepository(database.Conn())
	gradeRepo := repository.NewGradeRepository(database.Conn())
	timetableRepo := repository.NewTimetableRepository(database.Conn())
	notificationRepo := repository.NewNotificationRepository(database.Conn())
	newsRepo := repository.NewNewsRepository(database.Conn())

	// Initialize services
	authService := services.NewAuthService(userRepo, jwtSecret)
	classService := services.NewClassService(classRepo, userRepo)
	teacherService := services.NewTeacherService(teacherRepo, userRepo)
	studentService := services.NewStudentService(studentRepo, classRepo, userRepo)
	subjectService := services.NewSubjectService(subjectRepo, teacherRepo)
	userService := services.NewUserService(userRepo)
	parentService := services.NewParentService(parentRepo, studentRepo, userRepo)
	accessControl := services.NewAccessControl(classRepo, teacherRepo, parentRepo, timetableRepo)
	notificationService := services.NewNotificationService(notificationRepo, parentRepo)
	attendanceService := services.NewAttendanceService(attendanceRepo, studentRepo, classRepo, accessControl, notificationService)
	gradeService := services.NewGradeService(gradeRepo, studentRepo, subjectRepo, accessControl, notificationService)
	timetableService := services.NewTimetableService(timetableRepo, classRepo, subjectRepo, teacherRepo, studentRepo)
	reminderService := services.NewReminderService(attendanceRepo, notificationService)
	newsService := services.NewNewsService(newsRepo)

	// E-mail copies of notifications: real SMTP when configured, otherwise written to the log
	if host := os.Getenv("SMTP_HOST"); host != "" {
		notificationService.EnableEmail(mailer.NewSMTPMailer(mailer.SMTPConfig{
			Host: host, Port: os.Getenv("SMTP_PORT"),
			Username: os.Getenv("SMTP_USERNAME"), Password: os.Getenv("SMTP_PASSWORD"),
			From: os.Getenv("SMTP_FROM"),
		}), userRepo)
		logger.Info("E-mail notifications enabled", "smtp_host", host)
	} else if os.Getenv("EMAIL_LOG") == "true" {
		notificationService.EnableEmail(mailer.LogMailer{}, userRepo)
	}

	// Public self-registration is disabled, so the first administrator comes from configuration
	if err := userService.EnsureAdmin(context.Background(), os.Getenv("ADMIN_EMAIL"), os.Getenv("ADMIN_PASSWORD")); err != nil {
		logger.Error("Failed to create bootstrap admin", "error", err)
		os.Exit(1)
	}

	// Background job: remind homeroom teachers about unmarked attendance (default 15:00 server time)
	jobsCtx, stopJobs := context.WithCancel(context.Background())
	defer stopJobs()
	if hour := envInt("REMINDER_HOUR", 15); hour >= 0 && hour <= 23 {
		reminderService.Start(jobsCtx, hour)
	}

	// Password guessing protection: 10 login attempts per minute per IP (API and web share it)
	loginLimiter := appMiddleware.NewRateLimiter(envInt("LOGIN_RATE_LIMIT", 10), time.Minute)

	r, err := app.NewRouter(app.Deps{
		Auth: authService, Users: userService, Classes: classService, Teachers: teacherService,
		Students: studentService, Subjects: subjectService, Parents: parentService,
		Attendance: attendanceService, Grades: gradeService, Timetable: timetableService,
		Notifications: notificationService, Reminders: reminderService, News: newsService, Access: accessControl,
		LoginLimiter: loginLimiter, SecureCookie: os.Getenv("COOKIE_SECURE") == "true", AccessLog: true,
	})
	if err != nil {
		logger.Error("Failed to build the router", "error", err)
		os.Exit(1)
	}

	// Start server
	server := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	logger.Info("Starting server", "port", port)

	// Start server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server error", "error", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown error", "error", err)
		os.Exit(1)
	}

	logger.Info("Server stopped")
}

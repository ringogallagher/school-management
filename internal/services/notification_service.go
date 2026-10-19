package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/mzviagintsev/school-management/internal/models"
)

// Mailer sends e-mail copies of notifications (see internal/mailer)
type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// NotificationService stores in-app notifications and exposes them to users.
// Other services call NotifyStudentAndParents when something happens to a
// student (a grade is posted, an absence is recorded, attendance drops).
type NotificationService struct {
	notificationRepo NotificationRepository
	parentRepo       ParentRepository

	// optional e-mail delivery (nil = in-app only)
	mailer    Mailer
	userRepo  UserRepository
	syncEmail bool // tests send synchronously; production sends in the background
}

// NewNotificationService creates a new notification service
func NewNotificationService(notificationRepo NotificationRepository, parentRepo ParentRepository) *NotificationService {
	return &NotificationService{
		notificationRepo: notificationRepo,
		parentRepo:       parentRepo,
	}
}

// EnableEmail turns on e-mail copies of notifications
func (ns *NotificationService) EnableEmail(mailer Mailer, userRepo UserRepository) {
	ns.mailer = mailer
	ns.userRepo = userRepo
}

// NotifyStudentAndParents sends the same notification to a student and every parent linked to them.
// Failures are logged rather than returned: a failed notification must never undo the
// grade or attendance record that triggered it.
func (ns *NotificationService) NotifyStudentAndParents(ctx context.Context, student *models.Student, notifType, title, message string) {
	recipients := []int64{student.UserID}

	parentIDs, err := ns.parentRepo.ParentUserIDsForStudent(ctx, student.ID)
	if err != nil {
		slog.Warn("failed to load parents for notification", "student_id", student.ID, "error", err)
	} else {
		recipients = append(recipients, parentIDs...)
	}
	ns.NotifyUsers(ctx, recipients, notifType, title, message)
}

// NotifyUsers stores an in-app notification for each user and, when e-mail is enabled,
// sends an e-mail copy. Errors are logged, never returned.
func (ns *NotificationService) NotifyUsers(ctx context.Context, userIDs []int64, notifType, title, message string) {
	batch := make([]*models.Notification, 0, len(userIDs))
	for _, uid := range userIDs {
		batch = append(batch, &models.Notification{
			UserID:  uid,
			Type:    notifType,
			Title:   title,
			Message: message,
		})
	}
	if err := ns.notificationRepo.CreateBatch(ctx, batch); err != nil {
		slog.Warn("failed to create notifications", "type", notifType, "error", err)
	}

	if ns.mailer == nil || ns.userRepo == nil {
		return
	}
	send := func(ctx context.Context) {
		for _, uid := range userIDs {
			u, err := ns.userRepo.GetByID(ctx, uid)
			if err != nil || u == nil || !u.IsActive {
				continue
			}
			body := fmt.Sprintf("Hello %s,\n\n%s\n\n-- School Management System", u.FirstName, message)
			if err := ns.mailer.Send(ctx, u.Email, title, body); err != nil {
				slog.Warn("failed to send notification email", "user_id", uid, "error", err)
			}
		}
	}
	if ns.syncEmail {
		send(ctx)
		return
	}
	// Do not hold up the HTTP request (or depend on its context) while talking to SMTP
	go func() {
		bg, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		send(bg)
	}()
}

// ListForUser returns the user's notifications, newest first
func (ns *NotificationService) ListForUser(ctx context.Context, userID int64, unreadOnly bool, limit, offset int) ([]*models.Notification, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	items, err := ns.notificationRepo.ListByUser(ctx, userID, unreadOnly, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list notifications: %w", err)
	}
	return items, nil
}

// UnreadCount returns how many unread notifications the user has
func (ns *NotificationService) UnreadCount(ctx context.Context, userID int64) (int, error) {
	n, err := ns.notificationRepo.UnreadCount(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to count notifications: %w", err)
	}
	return n, nil
}

// MarkRead marks one of the user's notifications as read. A user can only
// mark their own notifications; anything else is reported as not found.
func (ns *NotificationService) MarkRead(ctx context.Context, userID, notificationID int64) error {
	return ns.notificationRepo.MarkRead(ctx, notificationID, userID)
}

// MarkAllRead marks all of the user's notifications as read and returns how many changed
func (ns *NotificationService) MarkAllRead(ctx context.Context, userID int64) (int64, error) {
	n, err := ns.notificationRepo.MarkAllRead(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to mark notifications read: %w", err)
	}
	return n, nil
}

// Announce broadcasts a system announcement to all users, or to one role
func (ns *NotificationService) Announce(ctx context.Context, req *models.AnnouncementRequest) (int64, error) {
	title := strings.TrimSpace(req.Title)
	message := strings.TrimSpace(req.Message)
	if title == "" || message == "" {
		return 0, fmt.Errorf("title and message are required")
	}
	if len(title) > 200 {
		return 0, fmt.Errorf("title must be at most 200 characters")
	}
	if req.Role != "" && !isValidRole(req.Role) {
		return 0, fmt.Errorf("invalid role %q", req.Role)
	}
	n, err := ns.notificationRepo.Broadcast(ctx, req.Role, models.NotificationAnnouncement, title, message)
	if err != nil {
		return 0, fmt.Errorf("failed to send announcement: %w", err)
	}
	return n, nil
}

func isValidRole(r models.UserRole) bool {
	switch r {
	case models.RoleAdmin, models.RoleTeacher, models.RoleStudent, models.RoleParent:
		return true
	}
	return false
}

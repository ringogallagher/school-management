package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/mzviagintsev/school-management/internal/models"
)

// NotificationRepository handles notification database operations
type NotificationRepository struct {
	db *sql.DB
}

// NewNotificationRepository creates a new notification repository
func NewNotificationRepository(db *sql.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// CreateBatch inserts several notifications in one transaction
func (nr *NotificationRepository) CreateBatch(ctx context.Context, items []*models.Notification) error {
	if len(items) == 0 {
		return nil
	}
	tx, err := nr.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO notifications (user_id, type, title, message, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`)
	if err != nil {
		return fmt.Errorf("failed to prepare notification insert: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, n := range items {
		if err := stmt.QueryRowContext(ctx, n.UserID, n.Type, n.Title, n.Message, now).Scan(&n.ID, &n.CreatedAt); err != nil {
			return fmt.Errorf("failed to create notification: %w", err)
		}
	}
	return tx.Commit()
}

// ListByUser returns a user's notifications, newest first
func (nr *NotificationRepository) ListByUser(ctx context.Context, userID int64, unreadOnly bool, limit, offset int) ([]*models.Notification, error) {
	rows, err := nr.db.QueryContext(ctx, `
		SELECT id, user_id, type, title, message, is_read, read_at, created_at
		FROM notifications
		WHERE user_id = $1 AND (NOT $2 OR is_read = false)
		ORDER BY created_at DESC, id DESC
		LIMIT $3 OFFSET $4`, userID, unreadOnly, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list notifications: %w", err)
	}
	defer rows.Close()

	result := []*models.Notification{}
	for rows.Next() {
		n := &models.Notification{}
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Message, &n.IsRead, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}
		result = append(result, n)
	}
	return result, rows.Err()
}

// UnreadCount counts a user's unread notifications
func (nr *NotificationRepository) UnreadCount(ctx context.Context, userID int64) (int, error) {
	var n int
	err := nr.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false`, userID).Scan(&n)
	return n, err
}

// MarkRead marks one notification read; only succeeds if it belongs to the user
func (nr *NotificationRepository) MarkRead(ctx context.Context, id, userID int64) error {
	res, err := nr.db.ExecContext(ctx, `
		UPDATE notifications SET is_read = true, read_at = COALESCE(read_at, $1)
		WHERE id = $2 AND user_id = $3`, time.Now(), id, userID)
	if err != nil {
		return fmt.Errorf("failed to mark notification read: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("notification not found")
	}
	return nil
}

// MarkAllRead marks all of a user's unread notifications read
func (nr *NotificationRepository) MarkAllRead(ctx context.Context, userID int64) (int64, error) {
	res, err := nr.db.ExecContext(ctx, `
		UPDATE notifications SET is_read = true, read_at = $1
		WHERE user_id = $2 AND is_read = false`, time.Now(), userID)
	if err != nil {
		return 0, fmt.Errorf("failed to mark notifications read: %w", err)
	}
	return res.RowsAffected()
}

// Broadcast inserts one notification per active user (optionally only one role) in a single statement
func (nr *NotificationRepository) Broadcast(ctx context.Context, role models.UserRole, notifType, title, message string) (int64, error) {
	res, err := nr.db.ExecContext(ctx, `
		INSERT INTO notifications (user_id, type, title, message, created_at)
		SELECT id, $1, $2, $3, $4
		FROM users
		WHERE is_active AND deleted_at IS NULL AND ($5 = '' OR role::text = $5)`,
		notifType, title, message, time.Now(), string(role))
	if err != nil {
		return 0, fmt.Errorf("failed to broadcast notification: %w", err)
	}
	return res.RowsAffected()
}

package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/mzviagintsev/school-management/internal/models"
)

// NewsRepository handles school news database operations
type NewsRepository struct {
	db *sql.DB
}

// NewNewsRepository creates a new news repository
func NewNewsRepository(db *sql.DB) *NewsRepository {
	return &NewsRepository{db: db}
}

const newsSelect = `
	SELECT n.id, n.title, n.body, n.pinned, n.author_id,
	       TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')),
	       n.created_at, n.updated_at
	FROM news n
	LEFT JOIN users u ON u.id = n.author_id
`

func scanNews(row interface{ Scan(...any) error }) (*models.News, error) {
	n := &models.News{}
	var author sql.NullInt64
	if err := row.Scan(&n.ID, &n.Title, &n.Body, &n.Pinned, &author, &n.AuthorName, &n.CreatedAt, &n.UpdatedAt); err != nil {
		return nil, err
	}
	if author.Valid {
		id := author.Int64
		n.AuthorID = &id
	}
	return n, nil
}

// Create inserts a news item
func (nr *NewsRepository) Create(ctx context.Context, n *models.News) error {
	query := `
		INSERT INTO news (title, body, pinned, author_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		RETURNING id, created_at, updated_at
	`
	if err := nr.db.QueryRowContext(ctx, query, n.Title, n.Body, n.Pinned, n.AuthorID, time.Now()).
		Scan(&n.ID, &n.CreatedAt, &n.UpdatedAt); err != nil {
		return fmt.Errorf("failed to create news: %w", err)
	}
	return nil
}

// GetByID retrieves a news item (nil, nil if not found)
func (nr *NewsRepository) GetByID(ctx context.Context, id int64) (*models.News, error) {
	n, err := scanNews(nr.db.QueryRowContext(ctx, newsSelect+` WHERE n.id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get news: %w", err)
	}
	return n, nil
}

// Update changes the title, body and pinned flag of a news item
func (nr *NewsRepository) Update(ctx context.Context, n *models.News) error {
	now := time.Now()
	res, err := nr.db.ExecContext(ctx,
		`UPDATE news SET title = $1, body = $2, pinned = $3, updated_at = $4 WHERE id = $5`,
		n.Title, n.Body, n.Pinned, now, n.ID)
	if err != nil {
		return fmt.Errorf("failed to update news: %w", err)
	}
	if c, _ := res.RowsAffected(); c == 0 {
		return fmt.Errorf("news not found")
	}
	n.UpdatedAt = now
	return nil
}

// Delete removes a news item
func (nr *NewsRepository) Delete(ctx context.Context, id int64) error {
	res, err := nr.db.ExecContext(ctx, `DELETE FROM news WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete news: %w", err)
	}
	if c, _ := res.RowsAffected(); c == 0 {
		return fmt.Errorf("news not found")
	}
	return nil
}

// List returns news items, pinned first and then newest first
func (nr *NewsRepository) List(ctx context.Context, limit, offset int) ([]*models.News, error) {
	rows, err := nr.db.QueryContext(ctx, newsSelect+` ORDER BY n.pinned DESC, n.created_at DESC, n.id DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list news: %w", err)
	}
	defer rows.Close()

	items := []*models.News{}
	for rows.Next() {
		n, err := scanNews(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan news: %w", err)
		}
		items = append(items, n)
	}
	return items, rows.Err()
}

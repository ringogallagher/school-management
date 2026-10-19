package services

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/mzviagintsev/school-management/internal/models"
)

const (
	maxNewsTitle = 200
	maxNewsBody  = 10000
)

// NewsService manages the school news shown on the home page.
// Reading is public; only administrators publish, edit and delete.
type NewsService struct {
	repo NewsRepository
}

// NewNewsService creates a new news service
func NewNewsService(repo NewsRepository) *NewsService {
	return &NewsService{repo: repo}
}

func validateNews(req *models.NewsRequest) (title, body string, err error) {
	title = strings.TrimSpace(req.Title)
	body = strings.TrimSpace(req.Body)
	switch {
	case title == "":
		return "", "", fmt.Errorf("title is required")
	case utf8.RuneCountInString(title) > maxNewsTitle:
		return "", "", fmt.Errorf("title must be at most %d characters", maxNewsTitle)
	case body == "":
		return "", "", fmt.Errorf("text is required")
	case utf8.RuneCountInString(body) > maxNewsBody:
		return "", "", fmt.Errorf("text must be at most %d characters", maxNewsBody)
	}
	return title, body, nil
}

func requireAdmin(actor Actor) error {
	if actor.Role != models.RoleAdmin {
		return ErrForbidden
	}
	return nil
}

// Create publishes a news item (admin only)
func (ns *NewsService) Create(ctx context.Context, actor Actor, req *models.NewsRequest) (*models.News, error) {
	if err := requireAdmin(actor); err != nil {
		return nil, err
	}
	title, body, err := validateNews(req)
	if err != nil {
		return nil, err
	}
	author := actor.UserID
	n := &models.News{Title: title, Body: body, Pinned: req.Pinned, AuthorID: &author}
	if err := ns.repo.Create(ctx, n); err != nil {
		return nil, fmt.Errorf("failed to publish news: %w", err)
	}
	return ns.Get(ctx, n.ID)
}

// Update replaces the title, text and pinned flag of a news item (admin only)
func (ns *NewsService) Update(ctx context.Context, actor Actor, id int64, req *models.NewsRequest) (*models.News, error) {
	if err := requireAdmin(actor); err != nil {
		return nil, err
	}
	title, body, err := validateNews(req)
	if err != nil {
		return nil, err
	}
	n, err := ns.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	n.Title, n.Body, n.Pinned = title, body, req.Pinned
	if err := ns.repo.Update(ctx, n); err != nil {
		return nil, fmt.Errorf("failed to update news: %w", err)
	}
	return n, nil
}

// Delete removes a news item (admin only)
func (ns *NewsService) Delete(ctx context.Context, actor Actor, id int64) error {
	if err := requireAdmin(actor); err != nil {
		return err
	}
	if _, err := ns.Get(ctx, id); err != nil {
		return err
	}
	if err := ns.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete news: %w", err)
	}
	return nil
}

// Get returns one news item; it is public
func (ns *NewsService) Get(ctx context.Context, id int64) (*models.News, error) {
	n, err := ns.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get news: %w", err)
	}
	if n == nil {
		return nil, fmt.Errorf("news: %w", ErrNotFound)
	}
	return n, nil
}

// List returns the news feed (pinned first, then newest first); it is public
func (ns *NewsService) List(ctx context.Context, limit, offset int) ([]*models.News, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	items, err := ns.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list news: %w", err)
	}
	if items == nil {
		items = []*models.News{}
	}
	return items, nil
}

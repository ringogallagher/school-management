package services

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/mzviagintsev/school-management/internal/models"
)

// MockNewsRepository is an in-memory NewsRepository
type MockNewsRepository struct {
	items  map[int64]*models.News
	nextID int64
	clock  time.Time
}

func NewMockNewsRepository() *MockNewsRepository {
	return &MockNewsRepository{items: map[int64]*models.News{}, nextID: 1, clock: time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)}
}

func (m *MockNewsRepository) Create(ctx context.Context, n *models.News) error {
	n.ID = m.nextID
	m.nextID++
	m.clock = m.clock.Add(time.Hour) // every item is a little newer than the previous one
	n.CreatedAt, n.UpdatedAt = m.clock, m.clock
	n.AuthorName = "Author"
	cp := *n
	m.items[n.ID] = &cp
	return nil
}
func (m *MockNewsRepository) GetByID(ctx context.Context, id int64) (*models.News, error) {
	if n, ok := m.items[id]; ok {
		cp := *n
		return &cp, nil
	}
	return nil, nil
}
func (m *MockNewsRepository) Update(ctx context.Context, n *models.News) error {
	cp := *n
	m.items[n.ID] = &cp
	return nil
}
func (m *MockNewsRepository) Delete(ctx context.Context, id int64) error {
	delete(m.items, id)
	return nil
}
func (m *MockNewsRepository) List(ctx context.Context, limit, offset int) ([]*models.News, error) {
	var all []*models.News
	for _, n := range m.items {
		cp := *n
		all = append(all, &cp)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Pinned != all[j].Pinned {
			return all[i].Pinned
		}
		return all[i].CreatedAt.After(all[j].CreatedAt)
	})
	if offset > len(all) {
		offset = len(all)
	}
	all = all[offset:]
	if limit < len(all) {
		all = all[:limit]
	}
	return all, nil
}

var (
	newsAdmin   = Actor{UserID: 1, Role: models.RoleAdmin}
	newsTeacher = Actor{UserID: 2, Role: models.RoleTeacher}
)

func TestNews_OnlyAdminCanPublishEditDelete(t *testing.T) {
	svc := NewNewsService(NewMockNewsRepository())
	ctx := context.Background()
	req := &models.NewsRequest{Title: "Open day", Body: "Come and see the school."}

	if _, err := svc.Create(ctx, newsTeacher, req); !errors.Is(err, ErrForbidden) {
		t.Errorf("teacher create: expected ErrForbidden, got %v", err)
	}
	n, err := svc.Create(ctx, newsAdmin, req)
	if err != nil {
		t.Fatalf("admin create failed: %v", err)
	}
	if n.AuthorID == nil || *n.AuthorID != newsAdmin.UserID {
		t.Errorf("expected the admin recorded as author, got %v", n.AuthorID)
	}
	if _, err := svc.Update(ctx, newsTeacher, n.ID, req); !errors.Is(err, ErrForbidden) {
		t.Errorf("teacher update: expected ErrForbidden, got %v", err)
	}
	if err := svc.Delete(ctx, newsTeacher, n.ID); !errors.Is(err, ErrForbidden) {
		t.Errorf("teacher delete: expected ErrForbidden, got %v", err)
	}
	if err := svc.Delete(ctx, newsAdmin, n.ID); err != nil {
		t.Errorf("admin delete failed: %v", err)
	}
	if _, err := svc.Get(ctx, n.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
	if err := svc.Delete(ctx, newsAdmin, n.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound deleting twice, got %v", err)
	}
}

func TestNews_Validation(t *testing.T) {
	svc := NewNewsService(NewMockNewsRepository())
	cases := map[string]models.NewsRequest{
		"empty title":    {Title: "  ", Body: "text"},
		"empty body":     {Title: "Title", Body: ""},
		"title too long": {Title: strings.Repeat("a", maxNewsTitle+1), Body: "text"},
		"body too long":  {Title: "Title", Body: strings.Repeat("a", maxNewsBody+1)},
	}
	for name, req := range cases {
		req := req
		t.Run(name, func(t *testing.T) {
			if _, err := svc.Create(context.Background(), newsAdmin, &req); err == nil {
				t.Error("expected a validation error, got nil")
			}
		})
	}
	// multi-byte text counts characters, not bytes
	ok := models.NewsRequest{Title: strings.Repeat("я", maxNewsTitle), Body: "text"}
	if _, err := svc.Create(context.Background(), newsAdmin, &ok); err != nil {
		t.Errorf("a title of exactly %d characters should be accepted: %v", maxNewsTitle, err)
	}
}

func TestNews_FeedPinnedFirstThenNewest(t *testing.T) {
	svc := NewNewsService(NewMockNewsRepository())
	ctx := context.Background()
	for _, r := range []models.NewsRequest{
		{Title: "old", Body: "x"},
		{Title: "pinned", Body: "x", Pinned: true},
		{Title: "new", Body: "x"},
	} {
		r := r
		if _, err := svc.Create(ctx, newsAdmin, &r); err != nil {
			t.Fatal(err)
		}
	}
	items, err := svc.List(ctx, 0, 0) // 0 falls back to the default page size
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, n := range items {
		got = append(got, n.Title)
	}
	if strings.Join(got, ",") != "pinned,new,old" {
		t.Errorf("expected pinned,new,old, got %v", got)
	}
}

func TestNews_UpdateKeepsAuthorAndChangesText(t *testing.T) {
	svc := NewNewsService(NewMockNewsRepository())
	ctx := context.Background()
	n, _ := svc.Create(ctx, newsAdmin, &models.NewsRequest{Title: "Draft", Body: "v1"})
	updated, err := svc.Update(ctx, newsAdmin, n.ID, &models.NewsRequest{Title: " Final ", Body: "v2", Pinned: true})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if updated.Title != "Final" || updated.Body != "v2" || !updated.Pinned {
		t.Errorf("unexpected update result: %+v", updated)
	}
	if updated.AuthorID == nil || *updated.AuthorID != newsAdmin.UserID {
		t.Errorf("author should be unchanged, got %v", updated.AuthorID)
	}
	if _, err := svc.Update(ctx, newsAdmin, 999, &models.NewsRequest{Title: "x", Body: "y"}); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound for unknown id, got %v", err)
	}
}

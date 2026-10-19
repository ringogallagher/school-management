package memrepo

import (
	"context"
	"sort"
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

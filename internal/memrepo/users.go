package memrepo

import (
	"context"

	"github.com/mzviagintsev/school-management/internal/models"
)

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	users  map[int64]*models.User
	nextID int64
}

// NewMockUserRepository creates a new mock repository
func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users:  make(map[int64]*models.User),
		nextID: 1,
	}
}

// Create mock implementation
func (m *MockUserRepository) Create(ctx context.Context, user *models.User) error {
	user.ID = m.nextID
	m.users[user.ID] = user
	m.nextID++
	return nil
}

// GetByID mock implementation
func (m *MockUserRepository) GetByID(ctx context.Context, id int64) (*models.User, error) {
	if user, ok := m.users[id]; ok {
		return user, nil
	}
	return nil, nil
}

// GetByEmail mock implementation
func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, nil
}

// Update mock implementation
func (m *MockUserRepository) Update(ctx context.Context, user *models.User) error {
	if _, ok := m.users[user.ID]; ok {
		m.users[user.ID] = user
		return nil
	}
	return nil
}

// Delete mock implementation
func (m *MockUserRepository) Delete(ctx context.Context, id int64) error {
	delete(m.users, id)
	return nil
}

// GetByRole mock implementation
func (m *MockUserRepository) GetByRole(ctx context.Context, role models.UserRole, limit int, offset int) ([]*models.User, error) {
	var result []*models.User
	for _, user := range m.users {
		if user.Role == role {
			result = append(result, user)
		}
	}
	return result, nil
}

// Tests

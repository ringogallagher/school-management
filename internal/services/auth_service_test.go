package services

import (
	"context"
	"testing"

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

func TestRegister_Success(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := NewAuthService(mockRepo, "test-secret")

	req := &models.RegisterRequest{
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "John",
		LastName:  "Doe",
		Role:      models.RoleStudent,
	}

	user, err := authService.Register(context.Background(), req)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if user.Email != req.Email {
		t.Errorf("Expected email %s, got %s", req.Email, user.Email)
	}

	if user.FirstName != req.FirstName {
		t.Errorf("Expected first name %s, got %s", req.FirstName, user.FirstName)
	}

	if user.ID == 0 {
		t.Error("Expected user to have an ID")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := NewAuthService(mockRepo, "test-secret")

	req := &models.RegisterRequest{
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "John",
		LastName:  "Doe",
		Role:      models.RoleStudent,
	}

	// First registration should succeed
	_, err := authService.Register(context.Background(), req)
	if err != nil {
		t.Fatalf("First registration failed: %v", err)
	}

	// Second registration with same email should fail
	_, err = authService.Register(context.Background(), req)
	if err == nil {
		t.Error("Expected error for duplicate email, got nil")
	}
}

func TestRegister_MissingFields(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := NewAuthService(mockRepo, "test-secret")

	testCases := []struct {
		name string
		req  *models.RegisterRequest
	}{
		{
			name: "Missing email",
			req: &models.RegisterRequest{
				Email:     "",
				Password:  "password123",
				FirstName: "John",
			},
		},
		{
			name: "Missing password",
			req: &models.RegisterRequest{
				Email:     "test@example.com",
				Password:  "",
				FirstName: "John",
			},
		},
		{
			name: "Missing first name",
			req: &models.RegisterRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := authService.Register(context.Background(), tc.req)
			if err == nil {
				t.Error("Expected error for missing fields, got nil")
			}
		})
	}
}

func TestLogin_Success(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := NewAuthService(mockRepo, "test-secret")

	// Register a user first
	regReq := &models.RegisterRequest{
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "John",
		LastName:  "Doe",
		Role:      models.RoleStudent,
	}

	authService.Register(context.Background(), regReq)

	// Now test login
	loginReq := &models.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	authResp, err := authService.Login(context.Background(), loginReq)
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	if authResp.Token == "" {
		t.Error("Expected token to be non-empty")
	}

	if authResp.User.Email != loginReq.Email {
		t.Errorf("Expected email %s, got %s", loginReq.Email, authResp.User.Email)
	}
}

func TestLogin_InvalidPassword(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := NewAuthService(mockRepo, "test-secret")

	// Register a user
	regReq := &models.RegisterRequest{
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "John",
		LastName:  "Doe",
		Role:      models.RoleStudent,
	}

	authService.Register(context.Background(), regReq)

	// Try to login with wrong password
	loginReq := &models.LoginRequest{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}

	_, err := authService.Login(context.Background(), loginReq)
	if err == nil {
		t.Error("Expected error for invalid password, got nil")
	}
}

func TestValidateToken(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := NewAuthService(mockRepo, "test-secret")

	// Create a user and generate a token
	user := &models.User{
		ID:        1,
		Email:     "test@example.com",
		FirstName: "John",
		LastName:  "Doe",
		Role:      models.RoleStudent,
		IsActive:  true,
	}

	token, _, err := authService.GenerateToken(user)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Validate the token
	claims, err := authService.ValidateToken(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if claims.UserID != user.ID {
		t.Errorf("Expected user ID %d, got %d", user.ID, claims.UserID)
	}

	if claims.Email != user.Email {
		t.Errorf("Expected email %s, got %s", user.Email, claims.Email)
	}
}

func TestValidateToken_InvalidToken(t *testing.T) {
	mockRepo := NewMockUserRepository()
	authService := NewAuthService(mockRepo, "test-secret")

	// Try to validate an invalid token
	_, err := authService.ValidateToken("invalid.token.here")
	if err == nil {
		t.Error("Expected error for invalid token, got nil")
	}
}

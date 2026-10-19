package services

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/mzviagintsev/school-management/internal/models"
)

// AuthService handles authentication operations
type AuthService struct {
	userRepo  UserRepository
	jwtSecret string
}

// Claims represents JWT claims
type Claims struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// NewAuthService creates a new auth service
func NewAuthService(userRepo UserRepository, jwtSecret string) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
	}
}

// Register creates a new user account
func (as *AuthService) Register(ctx context.Context, req *models.RegisterRequest) (*models.User, error) {
	// Validate input
	if req.Email == "" || req.Password == "" || req.FirstName == "" {
		return nil, fmt.Errorf("email, password, and first name are required")
	}
	if !isValidRole(req.Role) {
		return nil, fmt.Errorf("role must be one of admin, teacher, student, parent")
	}
	if len(req.Password) < MinPasswordLength {
		return nil, fmt.Errorf("password must be at least %d characters", MinPasswordLength)
	}

	// Check if user already exists
	existing, err := as.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("user with this email already exists")
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &models.User{
		Email:        req.Email,
		PasswordHash: string(passwordHash),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Role:         req.Role,
		IsActive:     true,
	}

	if err := as.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// Login authenticates a user and returns a JWT token
func (as *AuthService) Login(ctx context.Context, req *models.LoginRequest) (*models.AuthResponse, error) {
	// Validate input
	if req.Email == "" || req.Password == "" {
		return nil, fmt.Errorf("email and password are required")
	}

	// Get user by email
	user, err := as.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Check if user is active
	if !user.IsActive {
		return nil, fmt.Errorf("user account is disabled")
	}

	// Compare password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Generate JWT token
	token, expiresAt, err := as.GenerateToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &models.AuthResponse{
		Token:     token,
		User:      user,
		ExpiresAt: expiresAt,
	}, nil
}

// GenerateToken generates a JWT token for a user
func (as *AuthService) GenerateToken(user *models.User) (string, time.Time, error) {
	expiresAt := time.Now().Add(24 * time.Hour)

	claims := Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   string(user.Role),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(as.jwtSecret))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, expiresAt, nil
}

// ValidateToken validates a JWT token and returns claims
func (as *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(as.jwtSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// ChangePassword changes a user's password
func (as *AuthService) ChangePassword(ctx context.Context, userID int64, oldPassword, newPassword string) error {
	// Get user
	user, err := as.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return fmt.Errorf("user not found")
	}
	if len(newPassword) < MinPasswordLength {
		return fmt.Errorf("new password must be at least %d characters", MinPasswordLength)
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return fmt.Errorf("invalid password")
	}

	// Hash new password
	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	user.PasswordHash = string(newPasswordHash)
	if err := as.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// GetUserByID returns a user by ID
func (as *AuthService) GetUserByID(ctx context.Context, userID int64) (*models.User, error) {
	user, err := as.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

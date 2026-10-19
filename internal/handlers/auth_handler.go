package handlers

import (
	"encoding/json"
	//"fmt"
	"net/http"

	"github.com/mzviagintsev/school-management/internal/middleware"
	"github.com/mzviagintsev/school-management/internal/models"
	"github.com/mzviagintsev/school-management/internal/services"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	authService *services.AuthService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Register handles user registration
// POST /api/auth/register
func (ah *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := ah.authService.Register(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, &models.SuccessResponse{
		Code:    http.StatusCreated,
		Message: "user registered successfully",
		Data:    user,
	})
}

// Login handles user login
// POST /api/auth/login
func (ah *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	authResp, err := ah.authService.Login(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, &models.SuccessResponse{
		Code:    http.StatusOK,
		Message: "login successful",
		Data:    authResp,
	})
}

// GetProfile returns current user profile
// GET /api/auth/profile
func (ah *AuthHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := ah.authService.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, &models.SuccessResponse{
		Code:    http.StatusOK,
		Message: "success",
		Data:    user,
	})
}

// ChangePassword handles password change
// POST /api/auth/change-password
func (ah *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := ah.authService.ChangePassword(r.Context(), claims.UserID, req.OldPassword, req.NewPassword); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, &models.SuccessResponse{
		Code:    http.StatusOK,
		Message: "password changed successfully",
	})
}

// Helper functions

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(&models.ErrorResponse{
		Code:    status,
		Message: message,
	})
}

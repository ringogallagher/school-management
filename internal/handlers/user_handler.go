package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/mzviagintsev/school-management/internal/models"
	"github.com/mzviagintsev/school-management/internal/services"
)

// UserHandler implements admin user management
type UserHandler struct {
	userService *services.UserService
}

// NewUserHandler creates a new user handler
func NewUserHandler(s *services.UserService) *UserHandler {
	return &UserHandler{userService: s}
}

// List GET /api/users?role=teacher&q=smith&limit=&offset=
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)
	q := r.URL.Query()
	users, err := h.userService.List(r.Context(), models.UserRole(q.Get("role")), q.Get("q"), limit, offset)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "success", users)
}

// GetByID GET /api/users/{id}
func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	user, err := h.userService.GetByID(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "success", user)
}

// Update PUT /api/users/{id}
func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFromRequest(r)
	id, err := urlID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	user, err := h.userService.Update(r.Context(), actor, id, &req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "user updated", user)
}

// ResetPassword POST /api/users/{id}/reset-password
func (h *UserHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	var req models.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.userService.ResetPassword(r.Context(), id, req.NewPassword); err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "password reset", nil)
}

// Delete DELETE /api/users/{id} (soft delete)
func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFromRequest(r)
	id, err := urlID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	if err := h.userService.Delete(r.Context(), actor, id); err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "user deleted", nil)
}

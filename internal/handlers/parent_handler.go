package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/mzviagintsev/school-management/internal/models"
	"github.com/mzviagintsev/school-management/internal/services"
)

// ParentHandler handles parent accounts and parent <-> student links
type ParentHandler struct {
	parentService *services.ParentService
}

// NewParentHandler creates a new parent handler
func NewParentHandler(s *services.ParentService) *ParentHandler {
	return &ParentHandler{parentService: s}
}

// Create POST /api/parents (admin)
func (h *ParentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateParentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	parent, err := h.parentService.Create(r.Context(), &req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeCreated(w, "parent created successfully", parent)
}

func parentAndStudentIDs(r *http.Request) (int64, int64, bool) {
	parentID, err1 := urlID(r, "id")
	studentID, err2 := urlID(r, "studentID")
	return parentID, studentID, err1 == nil && err2 == nil
}

// LinkStudent POST /api/parents/{id}/students/{studentID} (admin)
func (h *ParentHandler) LinkStudent(w http.ResponseWriter, r *http.Request) {
	parentID, studentID, ok := parentAndStudentIDs(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid parent or student id")
		return
	}
	if err := h.parentService.LinkStudent(r.Context(), parentID, studentID); err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "parent linked to student", nil)
}

// UnlinkStudent DELETE /api/parents/{id}/students/{studentID} (admin)
func (h *ParentHandler) UnlinkStudent(w http.ResponseWriter, r *http.Request) {
	parentID, studentID, ok := parentAndStudentIDs(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid parent or student id")
		return
	}
	if err := h.parentService.UnlinkStudent(r.Context(), parentID, studentID); err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "parent unlinked from student", nil)
}

// MyChildren GET /api/parents/me/children (parent)
func (h *ParentHandler) MyChildren(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	children, err := h.parentService.ListChildren(r.Context(), actor.UserID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "success", children)
}

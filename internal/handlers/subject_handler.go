package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/mzviagintsev/school-management/internal/models"
	"github.com/mzviagintsev/school-management/internal/services"
)

// SubjectHandler handles subject requests
type SubjectHandler struct {
	subjectService *services.SubjectService
}

// NewSubjectHandler creates a new subject handler
func NewSubjectHandler(s *services.SubjectService) *SubjectHandler {
	return &SubjectHandler{subjectService: s}
}

// Create POST /api/subjects
func (h *SubjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateSubjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	subject, err := h.subjectService.Create(r.Context(), &req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeCreated(w, "subject created successfully", subject)
}

// GetByID GET /api/subjects/{id}
func (h *SubjectHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid subject id")
		return
	}
	subject, err := h.subjectService.GetByID(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "success", subject)
}

// List GET /api/subjects
func (h *SubjectHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)
	subjects, err := h.subjectService.List(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, "success", subjects)
}

// Update PUT /api/subjects/{id}
func (h *SubjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid subject id")
		return
	}
	var req models.UpdateSubjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	subject, err := h.subjectService.Update(r.Context(), id, &req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "subject updated successfully", subject)
}

// Delete DELETE /api/subjects/{id}
func (h *SubjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid subject id")
		return
	}
	if err := h.subjectService.Delete(r.Context(), id); err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "subject deleted successfully", nil)
}

// TeacherSubjects GET /api/teachers/{id}/subjects
func (h *SubjectHandler) TeacherSubjects(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid teacher id")
		return
	}
	subjects, err := h.subjectService.TeacherSubjects(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "success", subjects)
}

// AssignToTeacher PUT /api/teachers/{id}/subjects {"subject_ids": [1, 2]}
func (h *SubjectHandler) AssignToTeacher(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid teacher id")
		return
	}
	var req models.AssignSubjectsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	subjects, err := h.subjectService.AssignToTeacher(r.Context(), id, req.SubjectIDs)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "subjects assigned", subjects)
}

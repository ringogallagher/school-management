package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/mzviagintsev/school-management/internal/models"
	"github.com/mzviagintsev/school-management/internal/services"
)

// TeacherHandler handles teacher management requests
type TeacherHandler struct {
	teacherService *services.TeacherService
}

// NewTeacherHandler creates a new teacher handler
func NewTeacherHandler(teacherService *services.TeacherService) *TeacherHandler {
	return &TeacherHandler{teacherService: teacherService}
}

// Create handles teacher creation (creates both the user account and the teacher profile)
// POST /api/teachers
func (th *TeacherHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTeacherRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	teacher, err := th.teacherService.Create(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, &models.SuccessResponse{
		Code:    http.StatusCreated,
		Message: "teacher created successfully",
		Data:    teacher,
	})
}

// GetByID handles fetching a single teacher profile
// GET /api/teachers/{id}
func (th *TeacherHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid teacher id")
		return
	}

	teacher, err := th.teacherService.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, &models.SuccessResponse{
		Code:    http.StatusOK,
		Message: "success",
		Data:    teacher,
	})
}

// List handles listing teachers
// GET /api/teachers
func (th *TeacherHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)

	teachers, err := th.teacherService.List(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, &models.SuccessResponse{
		Code:    http.StatusOK,
		Message: "success",
		Data:    teachers,
	})
}

// Update handles updating a teacher's department/specialization
// PUT /api/teachers/{id}
func (th *TeacherHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid teacher id")
		return
	}

	var req models.UpdateTeacherRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	teacher, err := th.teacherService.Update(r.Context(), id, &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, &models.SuccessResponse{
		Code:    http.StatusOK,
		Message: "teacher updated successfully",
		Data:    teacher,
	})
}

// Delete handles removing a teacher profile and deactivating the linked user account
// DELETE /api/teachers/{id}
func (th *TeacherHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid teacher id")
		return
	}

	if err := th.teacherService.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, &models.SuccessResponse{
		Code:    http.StatusOK,
		Message: "teacher deleted successfully",
	})
}

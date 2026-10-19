package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/mzviagintsev/school-management/internal/models"
	"github.com/mzviagintsev/school-management/internal/services"
)

// StudentHandler handles student management requests
type StudentHandler struct {
	studentService *services.StudentService
}

// NewStudentHandler creates a new student handler
func NewStudentHandler(studentService *services.StudentService) *StudentHandler {
	return &StudentHandler{studentService: studentService}
}

// Create handles student creation (creates both the user account and the student profile)
// POST /api/students
func (sh *StudentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateStudentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	student, err := sh.studentService.Create(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, &models.SuccessResponse{
		Code:    http.StatusCreated,
		Message: "student created successfully",
		Data:    student,
	})
}

// GetByID handles fetching a single student profile
// GET /api/students/{id}
func (sh *StudentHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid student id")
		return
	}

	student, err := sh.studentService.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, &models.SuccessResponse{
		Code:    http.StatusOK,
		Message: "success",
		Data:    student,
	})
}

// List handles listing students
// GET /api/students
func (sh *StudentHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)

	students, err := sh.studentService.List(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, &models.SuccessResponse{
		Code:    http.StatusOK,
		Message: "success",
		Data:    students,
	})
}

// ListByClass handles listing students enrolled in a specific class
// GET /api/students/class/{classID}
func (sh *StudentHandler) ListByClass(w http.ResponseWriter, r *http.Request) {
	classID, err := strconv.ParseInt(chi.URLParam(r, "classID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid class id")
		return
	}

	limit, offset := parsePagination(r)

	students, err := sh.studentService.ListByClass(r.Context(), classID, limit, offset)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, &models.SuccessResponse{
		Code:    http.StatusOK,
		Message: "success",
		Data:    students,
	})
}

// Update handles reassigning a student to a different class
// PUT /api/students/{id}
func (sh *StudentHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid student id")
		return
	}

	var req models.UpdateStudentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	student, err := sh.studentService.Update(r.Context(), id, &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, &models.SuccessResponse{
		Code:    http.StatusOK,
		Message: "student updated successfully",
		Data:    student,
	})
}

// Delete handles removing a student profile and deactivating the linked user account
// DELETE /api/students/{id}
func (sh *StudentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid student id")
		return
	}

	if err := sh.studentService.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, &models.SuccessResponse{
		Code:    http.StatusOK,
		Message: "student deleted successfully",
	})
}

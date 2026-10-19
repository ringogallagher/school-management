package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/mzviagintsev/school-management/internal/models"
	"github.com/mzviagintsev/school-management/internal/services"
)

// ClassHandler handles class management requests
type ClassHandler struct {
	classService *services.ClassService
}

// NewClassHandler creates a new class handler
func NewClassHandler(classService *services.ClassService) *ClassHandler {
	return &ClassHandler{classService: classService}
}

// Create handles class creation
// POST /api/classes
func (ch *ClassHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateClassRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	class, err := ch.classService.Create(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, &models.SuccessResponse{
		Code:    http.StatusCreated,
		Message: "class created successfully",
		Data:    class,
	})
}

// GetByID handles fetching a single class
// GET /api/classes/{id}
func (ch *ClassHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid class id")
		return
	}

	class, err := ch.classService.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, &models.SuccessResponse{
		Code:    http.StatusOK,
		Message: "success",
		Data:    class,
	})
}

// List handles listing classes
// GET /api/classes
func (ch *ClassHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)

	classes, err := ch.classService.List(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, &models.SuccessResponse{
		Code:    http.StatusOK,
		Message: "success",
		Data:    classes,
	})
}

// Update handles updating a class
// PUT /api/classes/{id}
func (ch *ClassHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid class id")
		return
	}

	var req models.UpdateClassRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	class, err := ch.classService.Update(r.Context(), id, &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, &models.SuccessResponse{
		Code:    http.StatusOK,
		Message: "class updated successfully",
		Data:    class,
	})
}

// Delete handles deleting a class
// DELETE /api/classes/{id}
func (ch *ClassHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid class id")
		return
	}

	if err := ch.classService.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, &models.SuccessResponse{
		Code:    http.StatusOK,
		Message: "class deleted successfully",
	})
}

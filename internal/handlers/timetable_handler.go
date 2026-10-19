package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/mzviagintsev/school-management/internal/models"
	"github.com/mzviagintsev/school-management/internal/services"
)

// TimetableHandler handles timetable requests
type TimetableHandler struct {
	timetableService *services.TimetableService
}

// NewTimetableHandler creates a new timetable handler
func NewTimetableHandler(s *services.TimetableService) *TimetableHandler {
	return &TimetableHandler{timetableService: s}
}

// Create POST /api/timetable (admin)
func (h *TimetableHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.TimetableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	entry, err := h.timetableService.Create(r.Context(), &req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeCreated(w, "timetable entry created", entry)
}

// Update PUT /api/timetable/{id} (admin)
func (h *TimetableHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid timetable id")
		return
	}
	var req models.TimetableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	entry, err := h.timetableService.Update(r.Context(), id, &req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "timetable entry updated", entry)
}

// Delete DELETE /api/timetable/{id} (admin)
func (h *TimetableHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid timetable id")
		return
	}
	if err := h.timetableService.Delete(r.Context(), id); err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "timetable entry deleted", nil)
}

// ByClass GET /api/timetable/class/{classID}
func (h *TimetableHandler) ByClass(w http.ResponseWriter, r *http.Request) {
	classID, err := urlID(r, "classID")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid class id")
		return
	}
	items, err := h.timetableService.ListByClass(r.Context(), classID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "success", items)
}

// ByTeacher GET /api/timetable/teacher/{teacherID}
func (h *TimetableHandler) ByTeacher(w http.ResponseWriter, r *http.Request) {
	teacherID, err := urlID(r, "teacherID")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid teacher id")
		return
	}
	items, err := h.timetableService.ListByTeacher(r.Context(), teacherID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "success", items)
}

// Mine GET /api/timetable/me (teacher or student)
func (h *TimetableHandler) Mine(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFromRequest(r)
	items, err := h.timetableService.ListForActor(r.Context(), actor)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "success", items)
}

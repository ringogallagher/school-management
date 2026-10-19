package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/mzviagintsev/school-management/internal/models"
	"github.com/mzviagintsev/school-management/internal/services"
)

// GradeHandler handles grade requests
type GradeHandler struct {
	gradeService *services.GradeService
}

// NewGradeHandler creates a new grade handler
func NewGradeHandler(s *services.GradeService) *GradeHandler {
	return &GradeHandler{gradeService: s}
}

// Record POST /api/grades (admin, teacher)
func (h *GradeHandler) Record(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFromRequest(r)
	var req models.RecordGradeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	grade, err := h.gradeService.Record(r.Context(), actor, &req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeCreated(w, "grade recorded", grade)
}

// Update PUT /api/grades/{id} (admin, or the teacher who recorded it)
func (h *GradeHandler) Update(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFromRequest(r)
	id, err := urlID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid grade id")
		return
	}
	var req models.UpdateGradeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	grade, err := h.gradeService.Update(r.Context(), actor, id, &req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "grade updated", grade)
}

// Delete DELETE /api/grades/{id} (admin, or the teacher who recorded it)
func (h *GradeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFromRequest(r)
	id, err := urlID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid grade id")
		return
	}
	if err := h.gradeService.Delete(r.Context(), actor, id); err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "grade deleted", nil)
}

// StudentReport GET /api/grades/student/{studentID}?term= (any role, access-checked)
func (h *GradeHandler) StudentReport(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFromRequest(r)
	studentID, err := urlID(r, "studentID")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid student id")
		return
	}
	report, err := h.gradeService.GetStudentReport(r.Context(), actor, studentID, r.URL.Query().Get("term"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "success", report)
}

// MyReport GET /api/grades/me?term= (student)
func (h *GradeHandler) MyReport(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFromRequest(r)
	report, err := h.gradeService.GetMyReport(r.Context(), actor, r.URL.Query().Get("term"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "success", report)
}

// ClassSubject GET /api/grades/class/{classID}/subject/{subjectID}?term= (admin, teacher)
func (h *GradeHandler) ClassSubject(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFromRequest(r)
	classID, err1 := urlID(r, "classID")
	subjectID, err2 := urlID(r, "subjectID")
	if err1 != nil || err2 != nil {
		writeError(w, http.StatusBadRequest, "invalid class or subject id")
		return
	}
	grades, err := h.gradeService.GetClassSubjectGrades(r.Context(), actor, classID, subjectID, r.URL.Query().Get("term"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "success", grades)
}

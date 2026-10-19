package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/mzviagintsev/school-management/internal/models"
	"github.com/mzviagintsev/school-management/internal/services"
)

// AttendanceHandler handles attendance requests
type AttendanceHandler struct {
	attendanceService *services.AttendanceService
}

// NewAttendanceHandler creates a new attendance handler
func NewAttendanceHandler(s *services.AttendanceService) *AttendanceHandler {
	return &AttendanceHandler{attendanceService: s}
}

// Mark POST /api/attendance (admin, teacher)
func (h *AttendanceHandler) Mark(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req models.MarkAttendanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	records, err := h.attendanceService.MarkAttendance(r.Context(), actor, &req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeCreated(w, "attendance recorded", records)
}

// ClassSheet GET /api/attendance/class/{classID}?date=YYYY-MM-DD (admin, teacher)
func (h *AttendanceHandler) ClassSheet(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFromRequest(r)
	classID, err := urlID(r, "classID")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid class id")
		return
	}
	records, err := h.attendanceService.GetClassAttendance(r.Context(), actor, classID, r.URL.Query().Get("date"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "success", records)
}

// ClassStatistics GET /api/attendance/class/{classID}/stats?from=&to= (admin, teacher)
func (h *AttendanceHandler) ClassStatistics(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFromRequest(r)
	classID, err := urlID(r, "classID")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid class id")
		return
	}
	q := r.URL.Query()
	stats, err := h.attendanceService.GetClassStatistics(r.Context(), actor, classID, q.Get("from"), q.Get("to"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "success", stats)
}

// StudentReport GET /api/attendance/student/{studentID}?from=&to= (any role, access-checked)
func (h *AttendanceHandler) StudentReport(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFromRequest(r)
	studentID, err := urlID(r, "studentID")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid student id")
		return
	}
	q := r.URL.Query()
	report, err := h.attendanceService.GetStudentReport(r.Context(), actor, studentID, q.Get("from"), q.Get("to"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "success", report)
}

// MyReport GET /api/attendance/me?from=&to= (student)
func (h *AttendanceHandler) MyReport(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFromRequest(r)
	q := r.URL.Query()
	report, err := h.attendanceService.GetMyReport(r.Context(), actor, q.Get("from"), q.Get("to"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "success", report)
}

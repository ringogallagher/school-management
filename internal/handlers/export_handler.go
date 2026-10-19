package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mzviagintsev/school-management/internal/services"
)

// ExportHandler serves CSV downloads of reports. Access rules are the same as for the
// JSON reports because both go through the same services.
type ExportHandler struct {
	grades     *services.GradeService
	attendance *services.AttendanceService
}

// NewExportHandler creates a new export handler
func NewExportHandler(g *services.GradeService, a *services.AttendanceService) *ExportHandler {
	return &ExportHandler{grades: g, attendance: a}
}

func sendCSV(w http.ResponseWriter, kind string, id int64, rows [][]string) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, services.ExportFilename(kind, id, time.Now().Format("2006-01-02"))))
	w.Header().Set("Cache-Control", "no-store")
	_ = services.WriteCSV(w, rows)
}

// StudentGrades GET /api/grades/student/{studentID}/export.csv?term=
func (h *ExportHandler) StudentGrades(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFromRequest(r)
	id, err := urlID(r, "studentID")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid student id")
		return
	}
	rep, err := h.grades.GetStudentReport(r.Context(), actor, id, r.URL.Query().Get("term"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	sendCSV(w, "marks", id, services.StudentGradesRows(rep))
}

// MyGrades GET /api/grades/me/export.csv?term= (student)
func (h *ExportHandler) MyGrades(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFromRequest(r)
	rep, err := h.grades.GetMyReport(r.Context(), actor, r.URL.Query().Get("term"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	sendCSV(w, "marks", rep.StudentID, services.StudentGradesRows(rep))
}

// ClassGrades GET /api/grades/class/{classID}/subject/{subjectID}/export.csv?term= (admin, teacher)
func (h *ExportHandler) ClassGrades(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFromRequest(r)
	classID, err1 := urlID(r, "classID")
	subjectID, err2 := urlID(r, "subjectID")
	if err1 != nil || err2 != nil {
		writeError(w, http.StatusBadRequest, "invalid class or subject id")
		return
	}
	grades, err := h.grades.GetClassSubjectGrades(r.Context(), actor, classID, subjectID, r.URL.Query().Get("term"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	sendCSV(w, "class-marks", classID, services.ClassGradesRows(grades))
}

// StudentAttendance GET /api/attendance/student/{studentID}/export.csv?from=&to=
func (h *ExportHandler) StudentAttendance(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFromRequest(r)
	id, err := urlID(r, "studentID")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid student id")
		return
	}
	q := r.URL.Query()
	rep, err := h.attendance.GetStudentReport(r.Context(), actor, id, q.Get("from"), q.Get("to"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	sendCSV(w, "attendance", id, services.StudentAttendanceRows(rep))
}

// MyAttendance GET /api/attendance/me/export.csv?from=&to= (student)
func (h *ExportHandler) MyAttendance(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFromRequest(r)
	q := r.URL.Query()
	rep, err := h.attendance.GetMyReport(r.Context(), actor, q.Get("from"), q.Get("to"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	sendCSV(w, "attendance", rep.Summary.StudentID, services.StudentAttendanceRows(rep))
}

// ClassAttendance GET /api/attendance/class/{classID}/stats.csv?from=&to= (admin, teacher)
func (h *ExportHandler) ClassAttendance(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFromRequest(r)
	classID, err := urlID(r, "classID")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid class id")
		return
	}
	q := r.URL.Query()
	stats, err := h.attendance.GetClassStatistics(r.Context(), actor, classID, q.Get("from"), q.Get("to"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	sendCSV(w, "class-attendance", classID, services.ClassAttendanceStatsRows(stats))
}

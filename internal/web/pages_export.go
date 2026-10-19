package web

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mzviagintsev/school-management/internal/services"
)

// CSV downloads for the report pages. Access is checked by the same services as the HTML pages.

func (h *Handler) sendCSV(w http.ResponseWriter, kind string, id int64, rows [][]string) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, services.ExportFilename(kind, id, time.Now().Format("2006-01-02"))))
	w.Header().Set("Cache-Control", "no-store")
	_ = services.WriteCSV(w, rows)
}

func (h *Handler) exportError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	if s := statusFor(err); s != 0 {
		status = s
	}
	http.Error(w, errMessage(err), status)
}

func (h *Handler) studentGradesCSV(w http.ResponseWriter, r *http.Request) {
	id := pathInt(r, "id")
	rep, err := h.svc.Grades.GetStudentReport(r.Context(), actorOf(r), id, r.URL.Query().Get("term"))
	if err != nil {
		h.exportError(w, err)
		return
	}
	h.sendCSV(w, "marks", id, services.StudentGradesRows(rep))
}

func (h *Handler) myGradesCSV(w http.ResponseWriter, r *http.Request) {
	rep, err := h.svc.Grades.GetMyReport(r.Context(), actorOf(r), r.URL.Query().Get("term"))
	if err != nil {
		h.exportError(w, err)
		return
	}
	h.sendCSV(w, "marks", rep.StudentID, services.StudentGradesRows(rep))
}

func (h *Handler) studentAttendanceCSV(w http.ResponseWriter, r *http.Request) {
	id := pathInt(r, "id")
	q := r.URL.Query()
	rep, err := h.svc.Attendance.GetStudentReport(r.Context(), actorOf(r), id, q.Get("from"), q.Get("to"))
	if err != nil {
		h.exportError(w, err)
		return
	}
	h.sendCSV(w, "attendance", id, services.StudentAttendanceRows(rep))
}

func (h *Handler) myAttendanceCSV(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	rep, err := h.svc.Attendance.GetMyReport(r.Context(), actorOf(r), q.Get("from"), q.Get("to"))
	if err != nil {
		h.exportError(w, err)
		return
	}
	h.sendCSV(w, "attendance", rep.Summary.StudentID, services.StudentAttendanceRows(rep))
}

func (h *Handler) classAttendanceCSV(w http.ResponseWriter, r *http.Request) {
	classID := queryInt(r, "class")
	q := r.URL.Query()
	stats, err := h.svc.Attendance.GetClassStatistics(r.Context(), actorOf(r), classID, q.Get("from"), q.Get("to"))
	if err != nil {
		h.exportError(w, err)
		return
	}
	h.sendCSV(w, "class-attendance", classID, services.ClassAttendanceStatsRows(stats))
}

func (h *Handler) gradebookCSV(w http.ResponseWriter, r *http.Request) {
	classID, subjectID := queryInt(r, "class"), queryInt(r, "subject")
	grades, err := h.svc.Grades.GetClassSubjectGrades(r.Context(), actorOf(r), classID, subjectID, r.URL.Query().Get("term"))
	if err != nil {
		h.exportError(w, err)
		return
	}
	h.sendCSV(w, "class-marks", classID, services.ClassGradesRows(grades))
}

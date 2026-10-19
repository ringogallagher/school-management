package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/mzviagintsev/school-management/internal/models"
	"github.com/mzviagintsev/school-management/internal/services"
)

// pickClass returns the requested class if the actor may use it, otherwise the first available one
func pickClass(classes []*models.Class, requested int64) *models.Class {
	for _, c := range classes {
		if c.ID == requested {
			return c
		}
	}
	if len(classes) > 0 {
		return classes[0]
	}
	return nil
}

// ---------- attendance sheet ----------

type sheetRow struct {
	Student *models.StudentWithUser
	Status  models.AttendanceStatus
	Remarks string
	Marked  bool
}

type sheetData struct {
	Classes  []*models.Class
	Class    *models.Class
	Date     string
	Rows     []sheetRow
	Statuses []models.AttendanceStatus
	Marked   int
}

var allStatuses = []models.AttendanceStatus{
	models.AttendancePresent, models.AttendanceAbsent, models.AttendanceLate, models.AttendanceExcused,
}

func (h *Handler) loadSheet(ctx context.Context, actor services.Actor, classID int64, date string) (*sheetData, error) {
	d := &sheetData{Classes: h.teachableClasses(ctx, actor), Date: date, Statuses: allStatuses}
	d.Class = pickClass(d.Classes, classID)
	if d.Class == nil {
		return d, nil
	}
	students, err := h.svc.Students.ListByClass(ctx, d.Class.ID, 100, 0)
	if err != nil {
		return d, err
	}
	existing, err := h.svc.Attendance.GetClassAttendance(ctx, actor, d.Class.ID, date)
	if err != nil {
		return d, err
	}
	byStudent := map[int64]*models.AttendanceWithStudent{}
	for _, a := range existing {
		byStudent[a.StudentID] = a
	}
	for _, s := range students {
		row := sheetRow{Student: s, Status: models.AttendancePresent}
		if a, ok := byStudent[s.ID]; ok {
			row.Status, row.Remarks, row.Marked = a.Status, a.Remarks, true
			d.Marked++
		}
		d.Rows = append(d.Rows, row)
	}
	return d, nil
}

func (h *Handler) attendanceSheet(w http.ResponseWriter, r *http.Request) {
	p := h.newPage(r, "Attendance", "attendance")
	date := r.URL.Query().Get("date")
	if date == "" {
		date = p.Today.Format("2006-01-02")
	}
	d, err := h.loadSheet(r.Context(), actorOf(r), queryInt(r, "class"), date)
	if err != nil {
		p.Error = errMessage(err)
	}
	p.Data = d
	h.page(w, "attendance", p)
}

func (h *Handler) attendanceSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	classID := formInt(r, "class_id")
	date := r.FormValue("date")
	req := &models.MarkAttendanceRequest{ClassID: classID, Date: date}
	for _, idStr := range r.Form["student"] {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			continue
		}
		req.Records = append(req.Records, models.AttendanceEntry{
			StudentID: id,
			Status:    models.AttendanceStatus(r.FormValue("status_" + idStr)),
			Remarks:   strings.TrimSpace(r.FormValue("remarks_" + idStr)),
		})
	}
	saved, err := h.svc.Attendance.MarkAttendance(r.Context(), actorOf(r), req)
	if err != nil {
		p := h.newPage(r, "Attendance", "attendance")
		p.Error = errMessage(err)
		p.Data, _ = h.loadSheet(r.Context(), actorOf(r), classID, date)
		h.page(w, "attendance", p)
		return
	}
	redirect(w, r, fmt.Sprintf("/attendance?class=%d&date=%s", classID, date),
		fmt.Sprintf("Attendance saved for %d students", len(saved)))
}

func (h *Handler) attendanceStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor := actorOf(r)
	p := h.newPage(r, "Attendance statistics", "attendance")
	q := r.URL.Query()
	classes := h.teachableClasses(ctx, actor)
	class := pickClass(classes, queryInt(r, "class"))
	data := map[string]any{"Classes": classes, "Class": class, "From": q.Get("from"), "To": q.Get("to")}
	if class != nil {
		stats, err := h.svc.Attendance.GetClassStatistics(ctx, actor, class.ID, q.Get("from"), q.Get("to"))
		if err != nil {
			p.Error = errMessage(err)
		}
		data["Stats"] = stats
	}
	p.Data = data
	h.page(w, "attendance_stats", p)
}

// ---------- gradebook ----------

type gradebookRow struct {
	Student *models.StudentWithUser
	Grades  []*models.GradeWithDetails
	Average float64
}

type gradebookData struct {
	Classes  []*models.Class
	Class    *models.Class
	Subjects []*models.Subject
	Subject  *models.Subject
	Term     string
	Rows     []gradebookRow
	Me       int64
	IsAdmin  bool
}

func (h *Handler) loadGradebook(ctx context.Context, actor services.Actor, classID, subjectID int64, term string) (*gradebookData, error) {
	d := &gradebookData{Classes: h.teachableClasses(ctx, actor), Term: term, Me: actor.UserID, IsAdmin: actor.Role == models.RoleAdmin}
	d.Class = pickClass(d.Classes, classID)
	d.Subjects, _ = h.svc.Subjects.List(ctx, 100, 0)
	for _, s := range d.Subjects {
		if s.ID == subjectID || d.Subject == nil {
			d.Subject = s
		}
		if s.ID == subjectID {
			break
		}
	}
	if d.Class == nil || d.Subject == nil {
		return d, nil
	}
	students, err := h.svc.Students.ListByClass(ctx, d.Class.ID, 100, 0)
	if err != nil {
		return d, err
	}
	grades, err := h.svc.Grades.GetClassSubjectGrades(ctx, actor, d.Class.ID, d.Subject.ID, term)
	if err != nil {
		return d, err
	}
	byStudent := map[int64][]*models.GradeWithDetails{}
	for _, g := range grades {
		byStudent[g.StudentID] = append(byStudent[g.StudentID], g)
	}
	for _, s := range students {
		row := gradebookRow{Student: s, Grades: byStudent[s.ID]}
		if n := len(row.Grades); n > 0 {
			sum := 0.0
			for _, g := range row.Grades {
				sum += g.Marks
			}
			row.Average = sum / float64(n)
		}
		d.Rows = append(d.Rows, row)
	}
	return d, nil
}

func gradebookURL(classID, subjectID int64, term string) string {
	return fmt.Sprintf("/gradebook?class=%d&subject=%d&term=%s", classID, subjectID, term)
}

func (h *Handler) gradebook(w http.ResponseWriter, r *http.Request) {
	p := h.newPage(r, "Gradebook", "gradebook")
	term := strings.TrimSpace(r.URL.Query().Get("term"))
	if term == "" {
		term = "Q1"
	}
	d, err := h.loadGradebook(r.Context(), actorOf(r), queryInt(r, "class"), queryInt(r, "subject"), term)
	if err != nil {
		p.Error = errMessage(err)
	}
	p.Data = d
	h.page(w, "gradebook", p)
}

func (h *Handler) gradebookError(w http.ResponseWriter, r *http.Request, err error) {
	p := h.newPage(r, "Gradebook", "gradebook")
	p.Error = errMessage(err)
	p.Data, _ = h.loadGradebook(r.Context(), actorOf(r), formInt(r, "class_id"), formInt(r, "subject_id"), r.FormValue("term"))
	h.page(w, "gradebook", p)
}

func (h *Handler) gradeAdd(w http.ResponseWriter, r *http.Request) {
	marks, err := strconv.ParseFloat(strings.TrimSpace(r.FormValue("marks")), 64)
	if err != nil {
		h.gradebookError(w, r, fmt.Errorf("choose a mark from 1 to 5"))
		return
	}
	g, err := h.svc.Grades.Record(r.Context(), actorOf(r), &models.RecordGradeRequest{
		StudentID: formInt(r, "student_id"), SubjectID: formInt(r, "subject_id"),
		Marks: marks, Term: r.FormValue("term"),
	})
	if err != nil {
		h.gradebookError(w, r, err)
		return
	}
	redirect(w, r, gradebookURL(formInt(r, "class_id"), g.SubjectID, g.Term), "Grade saved")
}

func (h *Handler) gradeUpdate(w http.ResponseWriter, r *http.Request) {
	marks, err := strconv.ParseFloat(strings.TrimSpace(r.FormValue("marks")), 64)
	if err != nil {
		h.gradebookError(w, r, fmt.Errorf("choose a mark from 1 to 5"))
		return
	}
	if _, err := h.svc.Grades.Update(r.Context(), actorOf(r), pathInt(r, "id"), &models.UpdateGradeRequest{Marks: &marks}); err != nil {
		h.gradebookError(w, r, err)
		return
	}
	redirect(w, r, gradebookURL(formInt(r, "class_id"), formInt(r, "subject_id"), r.FormValue("term")), "Grade updated")
}

func (h *Handler) gradeDelete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Grades.Delete(r.Context(), actorOf(r), pathInt(r, "id")); err != nil {
		h.gradebookError(w, r, err)
		return
	}
	redirect(w, r, gradebookURL(formInt(r, "class_id"), formInt(r, "subject_id"), r.FormValue("term")), "Grade deleted")
}

// ---------- student report pages ----------

type reportHeader struct {
	StudentID int64
	Name      string
	Self      bool
}

func (h *Handler) studentHeader(ctx context.Context, studentID int64, self bool) reportHeader {
	hdr := reportHeader{StudentID: studentID, Self: self}
	if s, err := h.svc.Students.GetByID(ctx, studentID); err == nil {
		if u, err := h.svc.Auth.GetUserByID(ctx, s.UserID); err == nil {
			hdr.Name = strings.TrimSpace(u.FirstName + " " + u.LastName)
		}
	}
	return hdr
}

func (h *Handler) renderGrades(w http.ResponseWriter, r *http.Request, report *models.StudentGradeReport, err error, hdr reportHeader, nav string) {
	p := h.newPage(r, "Grades", nav)
	if err != nil {
		p.Error = errMessage(err)
		if errors.Is(err, services.ErrForbidden) {
			p.Status = http.StatusForbidden
		}
	}
	p.Data = map[string]any{"Report": report, "Rows": subjectRows(report), "Header": hdr, "Term": r.URL.Query().Get("term")}
	h.page(w, "student_grades", p)
}

func (h *Handler) studentGrades(w http.ResponseWriter, r *http.Request) {
	id := pathInt(r, "id")
	report, err := h.svc.Grades.GetStudentReport(r.Context(), actorOf(r), id, r.URL.Query().Get("term"))
	h.renderGrades(w, r, report, err, h.studentHeader(r.Context(), id, false), "children")
}

func (h *Handler) myGrades(w http.ResponseWriter, r *http.Request) {
	report, err := h.svc.Grades.GetMyReport(r.Context(), actorOf(r), r.URL.Query().Get("term"))
	h.renderGrades(w, r, report, err, reportHeader{Self: true}, "grades")
}

func (h *Handler) renderAttendance(w http.ResponseWriter, r *http.Request, report *models.StudentAttendanceReport, err error, hdr reportHeader, nav string) {
	p := h.newPage(r, "Attendance", nav)
	if err != nil {
		p.Error = errMessage(err)
		if errors.Is(err, services.ErrForbidden) {
			p.Status = http.StatusForbidden
		}
	}
	q := r.URL.Query()
	p.Data = map[string]any{"Report": report, "Header": hdr, "From": q.Get("from"), "To": q.Get("to"),
		"Classes": h.classNames(r.Context())}
	h.page(w, "student_attendance", p)
}

func (h *Handler) studentAttendance(w http.ResponseWriter, r *http.Request) {
	id := pathInt(r, "id")
	q := r.URL.Query()
	report, err := h.svc.Attendance.GetStudentReport(r.Context(), actorOf(r), id, q.Get("from"), q.Get("to"))
	h.renderAttendance(w, r, report, err, h.studentHeader(r.Context(), id, false), "children")
}

func (h *Handler) myAttendance(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	report, err := h.svc.Attendance.GetMyReport(r.Context(), actorOf(r), q.Get("from"), q.Get("to"))
	h.renderAttendance(w, r, report, err, reportHeader{Self: true}, "attendance")
}

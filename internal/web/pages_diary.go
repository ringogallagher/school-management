package web

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/mzviagintsev/school-management/internal/models"
	"github.com/mzviagintsev/school-management/internal/services"
)

// The diary is a week view of a pupil's timetable with the marks received on each day,
// the way a paper school diary (or an e-diary) is laid out.

type diaryLesson struct {
	Lesson *models.TimetableWithDetails
	Marks  []*models.GradeWithDetails
}

type diaryDay struct {
	Date    time.Time
	IsToday bool
	Lessons []diaryLesson
	// Other holds marks given that day in a subject that has no lesson on that weekday
	Other []*models.GradeWithDetails
}

type diaryPage struct {
	Header    reportHeader
	Base      string // path of this page, used for the week links
	Days      []diaryDay
	WeekLabel string
	Prev      string
	Next      string
	IsCurrent bool
}

const isoDay = "2006-01-02"

// weekStartOf returns the Monday of the week containing the date in raw (YYYY-MM-DD);
// an empty or invalid value means the week of today.
func weekStartOf(raw string, today time.Time) time.Time {
	d := today
	if parsed, err := time.ParseInLocation(isoDay, raw, today.Location()); err == nil {
		d = parsed
	}
	d = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, today.Location())
	return d.AddDate(0, 0, -((int(d.Weekday()) + 6) % 7))
}

// buildDiary lays out one week (Monday first). Saturday and Sunday only appear when they
// have lessons or marks. A mark goes next to the first lesson of its subject on the day it was given.
func buildDiary(weekStart time.Time, lessons []*models.TimetableWithDetails, grades []*models.GradeWithDetails, today time.Time) []diaryDay {
	byWeekday := map[int][]*models.TimetableWithDetails{}
	for _, l := range lessons {
		byWeekday[l.DayOfWeek] = append(byWeekday[l.DayOfWeek], l)
	}

	var days []diaryDay
	for i := 0; i < 7; i++ {
		date := weekStart.AddDate(0, 0, i)
		key := date.Format(isoDay)

		var dayMarks []*models.GradeWithDetails
		for _, g := range grades {
			if g.CreatedAt.Format(isoDay) == key {
				dayMarks = append(dayMarks, g)
			}
		}
		sort.SliceStable(dayMarks, func(a, b int) bool { return dayMarks[a].CreatedAt.Before(dayMarks[b].CreatedAt) })

		todays := append([]*models.TimetableWithDetails(nil), byWeekday[int(date.Weekday())]...)
		sort.SliceStable(todays, func(a, b int) bool { return todays[a].StartTime < todays[b].StartTime })

		if i >= 5 && len(todays) == 0 && len(dayMarks) == 0 {
			continue // empty weekend
		}

		day := diaryDay{Date: date, IsToday: key == today.Format(isoDay)}
		firstOfSubject := map[int64]int{}
		for _, l := range todays {
			if _, seen := firstOfSubject[l.SubjectID]; !seen {
				firstOfSubject[l.SubjectID] = len(day.Lessons)
			}
			day.Lessons = append(day.Lessons, diaryLesson{Lesson: l})
		}
		for _, g := range dayMarks {
			if idx, ok := firstOfSubject[g.SubjectID]; ok {
				day.Lessons[idx].Marks = append(day.Lessons[idx].Marks, g)
			} else {
				day.Other = append(day.Other, g)
			}
		}
		days = append(days, day)
	}
	return days
}

func (h *Handler) renderDiary(w http.ResponseWriter, r *http.Request, base string, report *models.StudentGradeReport, err error, hdr reportHeader, nav string) {
	p := h.newPage(r, "Diary", nav)
	p.Plain = true
	if err != nil {
		p.Error = errMessage(err)
		p.Status = statusFor(err)
		p.Data = diaryPage{Header: hdr, Base: base}
		h.page(w, "diary", p)
		return
	}

	weekStart := weekStartOf(r.URL.Query().Get("week"), p.Today)
	var lessons []*models.TimetableWithDetails
	if student, serr := h.svc.Students.GetByID(r.Context(), report.StudentID); serr == nil {
		lessons, _ = h.svc.Timetable.ListByClass(r.Context(), student.ClassID)
	}

	weekEnd := weekStart.AddDate(0, 0, 6)
	p.Data = diaryPage{
		Header:    hdr,
		Base:      base,
		Days:      buildDiary(weekStart, lessons, report.Grades, p.Today),
		WeekLabel: weekStart.Format("2 Jan") + " – " + weekEnd.Format("2 Jan 2006"),
		Prev:      weekStart.AddDate(0, 0, -7).Format(isoDay),
		Next:      weekStart.AddDate(0, 0, 7).Format(isoDay),
		IsCurrent: weekStart.Equal(weekStartOf("", p.Today)),
	}
	h.page(w, "diary", p)
}

func (h *Handler) myDiary(w http.ResponseWriter, r *http.Request) {
	report, err := h.svc.Grades.GetMyReport(r.Context(), actorOf(r), "")
	h.renderDiary(w, r, "/my/diary", report, err, reportHeader{Self: true}, "diary")
}

func (h *Handler) studentDiary(w http.ResponseWriter, r *http.Request) {
	id := pathInt(r, "id")
	report, err := h.svc.Grades.GetStudentReport(r.Context(), actorOf(r), id, "")
	h.renderDiary(w, r, fmt.Sprintf("/students/%d/diary", id), report, err, h.studentHeader(r.Context(), id, false), "children")
}

// statusFor maps a forbidden error to 403 so the error page is not served as 200
func statusFor(err error) int {
	if errors.Is(err, services.ErrForbidden) {
		return http.StatusForbidden
	}
	return 0
}

// latestMarks returns the n most recent marks, newest first
func latestMarks(grades []*models.GradeWithDetails, n int) []*models.GradeWithDetails {
	out := append([]*models.GradeWithDetails(nil), grades...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// subjectRow is one line of the "marks by subject" table: all marks in date order and the average
type subjectRow struct {
	Name    string
	Marks   []*models.GradeWithDetails
	Average float64
	Final   int
}

// subjectRows groups a report's marks by subject (alphabetical, like the report's averages)
func subjectRows(rep *models.StudentGradeReport) []subjectRow {
	if rep == nil {
		return nil
	}
	rows := make([]subjectRow, 0, len(rep.Subjects))
	index := map[int64]int{}
	for _, s := range rep.Subjects {
		index[s.SubjectID] = len(rows)
		rows = append(rows, subjectRow{Name: s.SubjectName, Average: s.Average, Final: s.Final})
	}
	for _, g := range rep.Grades {
		if i, ok := index[g.SubjectID]; ok {
			rows[i].Marks = append(rows[i].Marks, g)
		}
	}
	for i := range rows {
		m := rows[i].Marks
		sort.SliceStable(m, func(a, b int) bool { return m[a].CreatedAt.Before(m[b].CreatedAt) })
	}
	return rows
}

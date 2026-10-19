package services

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/mzviagintsev/school-management/internal/models"
)

// CSV export of reports. The row builders are shared by the JSON API and the
// web frontend, so both produce identical files.

// utf8BOM lets Excel open the file as UTF-8 (names with accents and Cyrillic stay readable)
const utf8BOM = "\xEF\xBB\xBF"

// safeCell neutralises spreadsheet formula injection: a cell that starts with = + - @ (or a
// tab / carriage return) would be executed as a formula by Excel, so it is prefixed with an apostrophe.
func safeCell(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

// WriteCSV writes rows as UTF-8 CSV (with a BOM) after neutralising formula injection
func WriteCSV(w io.Writer, rows [][]string) error {
	if _, err := io.WriteString(w, utf8BOM); err != nil {
		return err
	}
	cw := csv.NewWriter(w)
	for _, row := range rows {
		clean := make([]string, len(row))
		for i, c := range row {
			clean[i] = safeCell(c)
		}
		if err := cw.Write(clean); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func num(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

// StudentGradesRows lists a student's marks and then the average of every subject
func StudentGradesRows(rep *models.StudentGradeReport) [][]string {
	rows := [][]string{{"Date", "Subject", "Term", "Mark"}}
	for _, g := range rep.Grades {
		rows = append(rows, []string{g.CreatedAt.Format("2006-01-02"), g.SubjectName, g.Term, num(g.Marks)})
	}
	for _, s := range rep.Subjects {
		rows = append(rows, []string{"", s.SubjectName, "Average", num(s.Average)})
	}
	rows = append(rows, []string{"", "All subjects", "Average", num(rep.OverallAverage)})
	return rows
}

// StudentAttendanceRows lists a student's attendance records and the summary
func StudentAttendanceRows(rep *models.StudentAttendanceReport) [][]string {
	rows := [][]string{{"Date", "Status", "Remarks"}}
	for _, a := range rep.Records {
		rows = append(rows, []string{a.Date.Format("2006-01-02"), string(a.Status), a.Remarks})
	}
	s := rep.Summary
	rows = append(rows,
		[]string{"Summary", "Total days", strconv.Itoa(s.Total)},
		[]string{"", "Present", strconv.Itoa(s.Present)},
		[]string{"", "Late", strconv.Itoa(s.Late)},
		[]string{"", "Absent", strconv.Itoa(s.Absent)},
		[]string{"", "Excused", strconv.Itoa(s.Excused)},
		[]string{"", "Attendance %", num(s.Percentage)},
	)
	return rows
}

// ClassAttendanceStatsRows has one line per student of the class
func ClassAttendanceStatsRows(stats []*models.AttendanceSummary) [][]string {
	rows := [][]string{{"Last name", "First name", "Total", "Present", "Late", "Absent", "Excused", "Attendance %"}}
	for _, s := range stats {
		rows = append(rows, []string{s.LastName, s.FirstName, strconv.Itoa(s.Total), strconv.Itoa(s.Present),
			strconv.Itoa(s.Late), strconv.Itoa(s.Absent), strconv.Itoa(s.Excused), num(s.Percentage)})
	}
	return rows
}

// ClassGradesRows lists every mark given to a class in one subject
func ClassGradesRows(grades []*models.GradeWithDetails) [][]string {
	rows := [][]string{{"Last name", "First name", "Subject", "Term", "Mark", "Date"}}
	for _, g := range grades {
		rows = append(rows, []string{g.LastName, g.FirstName, g.SubjectName, g.Term, num(g.Marks), g.CreatedAt.Format("2006-01-02")})
	}
	return rows
}

// ExportFilename builds a safe download name such as marks-12-2026-10-13.csv
func ExportFilename(kind string, id int64, date string) string {
	return fmt.Sprintf("%s-%d-%s.csv", kind, id, date)
}

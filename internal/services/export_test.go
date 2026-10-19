package services

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/mzviagintsev/school-management/internal/models"
)

func TestWriteCSV_BOMAndEscaping(t *testing.T) {
	var buf bytes.Buffer
	rows := [][]string{{"a,b", `say "hi"`, "multi\nline", "Árpád"}}
	if err := WriteCSV(&buf, rows); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(buf.String(), utf8BOM) {
		t.Error("expected a UTF-8 BOM so Excel reads accents correctly")
	}
	got, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(buf.String(), utf8BOM))).ReadAll()
	if err != nil || len(got) != 1 || got[0][0] != "a,b" || got[0][1] != `say "hi"` || got[0][2] != "multi\nline" || got[0][3] != "Árpád" {
		t.Errorf("round trip failed: %v %v", got, err)
	}
}

func TestWriteCSV_FormulaInjection(t *testing.T) {
	var buf bytes.Buffer
	_ = WriteCSV(&buf, [][]string{{"=HYPERLINK(\"http://evil\")", "+1", "-2", "@SUM(A1)", "safe", ""}})
	got, _ := csv.NewReader(strings.NewReader(strings.TrimPrefix(buf.String(), utf8BOM))).ReadAll()
	for i, c := range got[0][:4] {
		if !strings.HasPrefix(c, "'") {
			t.Errorf("cell %d (%q) is not neutralised", i, c)
		}
	}
	if got[0][4] != "safe" || got[0][5] != "" {
		t.Errorf("harmless cells must stay unchanged: %v", got[0])
	}
}

func TestStudentGradesRows(t *testing.T) {
	d := time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC)
	rep := &models.StudentGradeReport{
		Grades:         []*models.GradeWithDetails{{Grade: models.Grade{Marks: 5, Term: "Q1", CreatedAt: d}, SubjectName: "Math"}},
		Subjects:       []models.SubjectAverage{{SubjectName: "Math", Average: 5}},
		OverallAverage: 5,
	}
	rows := StudentGradesRows(rep)
	if len(rows) != 4 || rows[1][0] != "2026-10-05" || rows[1][3] != "5" || rows[3][1] != "All subjects" {
		t.Errorf("unexpected rows: %v", rows)
	}
}

func TestStudentAttendanceAndStatsRows(t *testing.T) {
	d := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	rep := &models.StudentAttendanceReport{
		Summary: models.AttendanceSummary{Total: 2, Present: 1, Absent: 1, Percentage: 50},
		Records: []*models.Attendance{{Date: d, Status: models.AttendanceAbsent, Remarks: "ill"}},
	}
	rows := StudentAttendanceRows(rep)
	if rows[1][1] != "absent" || rows[1][2] != "ill" || rows[len(rows)-1][2] != "50" {
		t.Errorf("unexpected rows: %v", rows)
	}
	stats := ClassAttendanceStatsRows([]*models.AttendanceSummary{{LastName: "Toth", FirstName: "Bence", Total: 10, Present: 9, Late: 1, Percentage: 100}})
	if len(stats) != 2 || stats[1][0] != "Toth" || stats[1][5] != "0" || stats[1][7] != "100" {
		t.Errorf("unexpected stats rows: %v", stats)
	}
	grades := ClassGradesRows([]*models.GradeWithDetails{{Grade: models.Grade{Marks: 4, Term: "Q1", CreatedAt: d}, LastName: "Toth", FirstName: "Bence", SubjectName: "Math"}})
	if len(grades) != 2 || grades[1][4] != "4" {
		t.Errorf("unexpected class rows: %v", grades)
	}
}

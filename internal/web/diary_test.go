package web

import (
	"strings"
	"testing"
	"time"

	"github.com/mzviagintsev/school-management/internal/models"
)

var testDay = time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC) // a Wednesday

func TestWeekStartOf(t *testing.T) {
	cases := map[string]string{
		"":           "2026-10-12", // this week's Monday
		"2026-10-18": "2026-10-12", // Sunday belongs to the week that started on Monday the 12th
		"2026-10-19": "2026-10-19",
		"garbage":    "2026-10-12",
	}
	for in, want := range cases {
		if got := weekStartOf(in, testDay).Format(isoDay); got != want {
			t.Errorf("weekStartOf(%q) = %s, want %s", in, got, want)
		}
	}
}

func lesson(day int, start string, subject int64) *models.TimetableWithDetails {
	return &models.TimetableWithDetails{Timetable: models.Timetable{DayOfWeek: day, StartTime: start, SubjectID: subject}}
}

func mark(subject int64, at time.Time, v float64) *models.GradeWithDetails {
	return &models.GradeWithDetails{Grade: models.Grade{SubjectID: subject, Marks: v, CreatedAt: at}}
}

func TestBuildDiary(t *testing.T) {
	monday := weekStartOf("", testDay)
	lessons := []*models.TimetableWithDetails{
		lesson(1, "10:00", 2), lesson(1, "08:00", 1), // Monday, given out of order
		lesson(3, "08:00", 1), lesson(3, "09:00", 1), // Wednesday: two lessons of the same subject
	}
	grades := []*models.GradeWithDetails{
		mark(1, monday.Add(9*time.Hour), 5),                  // Monday maths
		mark(1, monday.AddDate(0, 0, 2).Add(9*time.Hour), 4), // Wednesday maths: goes to the first Wednesday lesson only
		mark(2, monday.AddDate(0, 0, 2).Add(9*time.Hour), 3), // Wednesday, but no such lesson that day
		mark(1, monday.AddDate(0, 0, -7), 2),                 // previous week: ignored
	}
	days := buildDiary(monday, lessons, grades, testDay)

	if len(days) != 5 {
		t.Fatalf("expected Mon-Fri only (empty weekend hidden), got %d days", len(days))
	}
	if days[0].Lessons[0].Lesson.StartTime != "08:00" {
		t.Errorf("lessons should be ordered by start time, got %s first", days[0].Lessons[0].Lesson.StartTime)
	}
	if len(days[0].Lessons[0].Marks) != 1 || days[0].Lessons[0].Marks[0].Marks != 5 {
		t.Errorf("Monday maths mark missing: %+v", days[0].Lessons[0])
	}
	wed := days[2]
	if !wed.IsToday {
		t.Error("Wednesday 14 Oct should be flagged as today")
	}
	if len(wed.Lessons[0].Marks) != 1 || len(wed.Lessons[1].Marks) != 0 {
		t.Errorf("a mark should sit with the first lesson of its subject only: %+v", wed.Lessons)
	}
	if len(wed.Other) != 1 || wed.Other[0].Marks != 3 {
		t.Errorf("a mark without a lesson that day should land in Other, got %+v", wed.Other)
	}
}

func TestBuildDiary_ShowsWeekendWithMarks(t *testing.T) {
	monday := weekStartOf("", testDay)
	saturday := monday.AddDate(0, 0, 5).Add(10 * time.Hour)
	days := buildDiary(monday, nil, []*models.GradeWithDetails{mark(1, saturday, 5)}, testDay)
	if len(days) != 6 || len(days[5].Other) != 1 {
		t.Errorf("expected Saturday to appear with its mark, got %d days", len(days))
	}
}

func TestSubjectRowsAndLatest(t *testing.T) {
	d1 := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	d2 := d1.AddDate(0, 0, 3)
	rep := &models.StudentGradeReport{
		Subjects: []models.SubjectAverage{{SubjectID: 1, SubjectName: "Art", Average: 4.5, Final: 5}},
		Grades:   []*models.GradeWithDetails{mark(1, d2, 5), mark(1, d1, 4)},
	}
	rows := subjectRows(rep)
	if len(rows) != 1 || rows[0].Marks[0].Marks != 4 || rows[0].Final != 5 {
		t.Errorf("marks should be in date order within a subject: %+v", rows)
	}
	if got := latestMarks(rep.Grades, 1); len(got) != 1 || got[0].Marks != 5 {
		t.Errorf("latestMarks should return the newest first, got %+v", got)
	}
}

func TestParagraphsAndSnippet(t *testing.T) {
	text := "First  paragraph\nstill first.\r\n\r\n\r\nSecond one."
	ps := paragraphs(text)
	if len(ps) != 2 || ps[1] != "Second one." {
		t.Errorf("unexpected paragraphs: %q", ps)
	}
	long := strings.Repeat("word ", 100)
	s := snippet(long, 50)
	if len([]rune(s)) > 51 || !strings.HasSuffix(s, "…") {
		t.Errorf("snippet not shortened properly: %q", s)
	}
	if got := snippet("short", 50); got != "short" {
		t.Errorf("short text must stay intact, got %q", got)
	}
}

func TestBandMatchesFivePointScale(t *testing.T) {
	band := funcMap["band"].(func(float64) string)
	cases := map[float64]string{5: "5", 4.5: "5", 4.49: "4", 3: "3", 2: "2", 1: "1", 0: "0"}
	for in, want := range cases {
		if got := band(in); got != want {
			t.Errorf("band(%v) = %s, want %s", in, got, want)
		}
	}
}

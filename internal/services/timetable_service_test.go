package services

import (
	"context"
	"strings"
	"testing"

	"github.com/mzviagintsev/school-management/internal/models"
)

func (f *schoolFixture) lesson(classID, teacherID int64, day int, start, end, room string) *models.TimetableRequest {
	return &models.TimetableRequest{
		ClassID: classID, SubjectID: f.math.ID, TeacherID: teacherID,
		DayOfWeek: day, StartTime: start, EndTime: end, RoomNumber: room,
	}
}

func TestTimetable_CreateValid(t *testing.T) {
	f := newSchoolFixture(t)
	e, err := f.timetableS.Create(context.Background(), f.lesson(f.classA.ID, f.homeroomProfile.ID, 1, "8:00", "08:45", "101"))
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if e.StartTime != "08:00" || e.EndTime != "08:45" {
		t.Errorf("expected normalised HH:MM times, got %s-%s", e.StartTime, e.EndTime)
	}
}

func TestTimetable_Validation(t *testing.T) {
	f := newSchoolFixture(t)
	ctx := context.Background()
	cases := map[string]*models.TimetableRequest{
		"bad day":          f.lesson(f.classA.ID, f.homeroomProfile.ID, 7, "08:00", "08:45", ""),
		"end before start": f.lesson(f.classA.ID, f.homeroomProfile.ID, 1, "09:00", "08:00", ""),
		"zero length":      f.lesson(f.classA.ID, f.homeroomProfile.ID, 1, "09:00", "09:00", ""),
		"bad time":         f.lesson(f.classA.ID, f.homeroomProfile.ID, 1, "9am", "10:00", ""),
		"unknown class":    f.lesson(999, f.homeroomProfile.ID, 1, "08:00", "08:45", ""),
		"unknown teacher":  f.lesson(f.classA.ID, 999, 1, "08:00", "08:45", ""),
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := f.timetableS.Create(ctx, req); err == nil {
				t.Error("expected an error, got nil")
			}
		})
	}
}

func TestTimetable_ConflictDetection(t *testing.T) {
	f := newSchoolFixture(t)
	ctx := context.Background()
	if _, err := f.timetableS.Create(ctx, f.lesson(f.classA.ID, f.homeroomProfile.ID, 1, "08:00", "08:45", "101")); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	cases := []struct {
		name     string
		req      *models.TimetableRequest
		conflict string // substring expected in the error; "" means no conflict
	}{
		{"same class overlaps", f.lesson(f.classA.ID, f.otherProfile.ID, 1, "08:30", "09:15", "202"), "class already has a lesson"},
		{"same teacher overlaps", f.lesson(f.classB.ID, f.homeroomProfile.ID, 1, "08:15", "08:30", "202"), "teacher already teaches"},
		{"same room overlaps", f.lesson(f.classB.ID, f.otherProfile.ID, 1, "08:40", "09:30", "101"), "room 101"},
		{"back-to-back is fine", f.lesson(f.classA.ID, f.homeroomProfile.ID, 1, "08:45", "09:30", "101"), ""},
		{"other day is fine", f.lesson(f.classA.ID, f.homeroomProfile.ID, 2, "08:00", "08:45", "101"), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := f.timetableS.Create(ctx, tc.req)
			if tc.conflict == "" {
				if err != nil {
					t.Errorf("expected no conflict, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.conflict) {
				t.Errorf("expected conflict containing %q, got %v", tc.conflict, err)
			}
		})
	}
}

func TestTimetable_UpdateIgnoresItself(t *testing.T) {
	f := newSchoolFixture(t)
	ctx := context.Background()
	e, err := f.timetableS.Create(ctx, f.lesson(f.classA.ID, f.homeroomProfile.ID, 1, "08:00", "08:45", "101"))
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	// Extending the same lesson by 5 minutes must not clash with itself
	if _, err := f.timetableS.Update(ctx, e.ID, f.lesson(f.classA.ID, f.homeroomProfile.ID, 1, "08:00", "08:50", "101")); err != nil {
		t.Errorf("expected update to succeed, got %v", err)
	}
}

func TestTimetable_ListForActor(t *testing.T) {
	f := newSchoolFixture(t)
	ctx := context.Background()
	_, _ = f.timetableS.Create(ctx, f.lesson(f.classA.ID, f.otherProfile.ID, 1, "08:00", "08:45", ""))
	_, _ = f.timetableS.Create(ctx, f.lesson(f.classB.ID, f.otherProfile.ID, 2, "08:00", "08:45", ""))

	teacherView, err := f.timetableS.ListForActor(ctx, actorOf(f.otherTeacher))
	if err != nil || len(teacherView) != 2 {
		t.Errorf("teacher should see 2 lessons, got %d (%v)", len(teacherView), err)
	}
	studentView, err := f.timetableS.ListForActor(ctx, actorOf(f.studentUserA))
	if err != nil || len(studentView) != 1 {
		t.Errorf("student A should see 1 lesson, got %d (%v)", len(studentView), err)
	}
	if _, err := f.timetableS.ListForActor(ctx, actorOf(f.parentUser)); err == nil {
		t.Error("expected error for parent personal timetable")
	}
}

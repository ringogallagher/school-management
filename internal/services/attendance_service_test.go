package services

import (
	"context"
	"errors"
	"testing"

	"github.com/mzviagintsev/school-management/internal/models"
)

func TestAttendance_HomeroomTeacherCanMark(t *testing.T) {
	f := newSchoolFixture(t)
	recs, err := f.attendance.MarkAttendance(context.Background(), actorOf(f.homeroomTeacher), &models.MarkAttendanceRequest{
		ClassID: f.classA.ID,
		Records: []models.AttendanceEntry{{StudentID: f.studentA.ID, Status: models.AttendancePresent}},
	})
	if err != nil {
		t.Fatalf("MarkAttendance failed: %v", err)
	}
	if len(recs) != 1 || !recs[0].Date.Equal(f.today) {
		t.Errorf("expected one record dated today, got %+v", recs)
	}
	if recs[0].RecordedBy == nil || *recs[0].RecordedBy != f.homeroomTeacher.ID {
		t.Error("expected recorded_by to be the marking teacher")
	}
}

func TestAttendance_DefaultsStatusToPresent(t *testing.T) {
	f := newSchoolFixture(t)
	recs, err := f.attendance.MarkAttendance(context.Background(), actorOf(f.admin), &models.MarkAttendanceRequest{
		ClassID: f.classA.ID,
		Records: []models.AttendanceEntry{{StudentID: f.studentA.ID}},
	})
	if err != nil {
		t.Fatalf("MarkAttendance failed: %v", err)
	}
	if recs[0].Status != models.AttendancePresent {
		t.Errorf("expected default status present, got %s", recs[0].Status)
	}
}

func TestAttendance_TeacherOfOtherClassIsForbidden(t *testing.T) {
	f := newSchoolFixture(t)
	_, err := f.attendance.MarkAttendance(context.Background(), actorOf(f.otherTeacher), &models.MarkAttendanceRequest{
		ClassID: f.classA.ID,
		Records: []models.AttendanceEntry{{StudentID: f.studentA.ID, Status: models.AttendancePresent}},
	})
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestAttendance_SubjectTeacherViaTimetableCanMark(t *testing.T) {
	f := newSchoolFixture(t)
	// otherTeacher is not homeroom of class A but teaches it a lesson
	_ = f.timetable.Create(context.Background(), &models.Timetable{
		ClassID: f.classA.ID, SubjectID: f.math.ID, TeacherID: f.otherProfile.ID,
		DayOfWeek: 1, StartTime: "08:00", EndTime: "08:45",
	})
	_, err := f.attendance.MarkAttendance(context.Background(), actorOf(f.otherTeacher), &models.MarkAttendanceRequest{
		ClassID: f.classA.ID,
		Records: []models.AttendanceEntry{{StudentID: f.studentA.ID, Status: models.AttendanceLate}},
	})
	if err != nil {
		t.Errorf("expected subject teacher to be allowed, got %v", err)
	}
}

func TestAttendance_ValidationErrors(t *testing.T) {
	f := newSchoolFixture(t)
	admin := actorOf(f.admin)
	cases := []struct {
		name string
		req  models.MarkAttendanceRequest
	}{
		{"missing class", models.MarkAttendanceRequest{Records: []models.AttendanceEntry{{StudentID: f.studentA.ID}}}},
		{"no records", models.MarkAttendanceRequest{ClassID: f.classA.ID}},
		{"future date", models.MarkAttendanceRequest{ClassID: f.classA.ID, Date: "2026-10-07",
			Records: []models.AttendanceEntry{{StudentID: f.studentA.ID}}}},
		{"bad date", models.MarkAttendanceRequest{ClassID: f.classA.ID, Date: "06.10.2026",
			Records: []models.AttendanceEntry{{StudentID: f.studentA.ID}}}},
		{"bad status", models.MarkAttendanceRequest{ClassID: f.classA.ID,
			Records: []models.AttendanceEntry{{StudentID: f.studentA.ID, Status: "sick"}}}},
		{"student from other class", models.MarkAttendanceRequest{ClassID: f.classA.ID,
			Records: []models.AttendanceEntry{{StudentID: f.studentB.ID}}}},
		{"duplicate student", models.MarkAttendanceRequest{ClassID: f.classA.ID,
			Records: []models.AttendanceEntry{{StudentID: f.studentA.ID}, {StudentID: f.studentA.ID}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := tc.req
			if _, err := f.attendance.MarkAttendance(context.Background(), admin, &req); err == nil {
				t.Error("expected an error, got nil")
			}
		})
	}
	if len(f.attendanceRp.records) != 0 {
		t.Error("invalid requests must not write any records")
	}
}

func TestAttendance_RemarkingSameDayOverwrites(t *testing.T) {
	f := newSchoolFixture(t)
	ctx := context.Background()
	mark := func(status models.AttendanceStatus) {
		if _, err := f.attendance.MarkAttendance(ctx, actorOf(f.admin), &models.MarkAttendanceRequest{
			ClassID: f.classA.ID, Records: []models.AttendanceEntry{{StudentID: f.studentA.ID, Status: status}},
		}); err != nil {
			t.Fatalf("mark failed: %v", err)
		}
	}
	mark(models.AttendanceAbsent)
	mark(models.AttendanceExcused)

	if len(f.attendanceRp.records) != 1 {
		t.Fatalf("expected a single record after re-marking, got %d", len(f.attendanceRp.records))
	}
	for _, r := range f.attendanceRp.records {
		if r.Status != models.AttendanceExcused {
			t.Errorf("expected overwritten status excused, got %s", r.Status)
		}
	}
}

func TestAttendance_AbsenceNotifiesStudentAndParent(t *testing.T) {
	f := newSchoolFixture(t)
	_, err := f.attendance.MarkAttendance(context.Background(), actorOf(f.admin), &models.MarkAttendanceRequest{
		ClassID: f.classA.ID,
		Records: []models.AttendanceEntry{{StudentID: f.studentA.ID, Status: models.AttendanceAbsent}},
	})
	if err != nil {
		t.Fatalf("MarkAttendance failed: %v", err)
	}
	if f.notifications.countFor(f.studentUserA.ID, models.NotificationAbsence) != 1 {
		t.Error("expected student to be notified of the absence")
	}
	if f.notifications.countFor(f.parentUser.ID, models.NotificationAbsence) != 1 {
		t.Error("expected parent to be notified of the absence")
	}
}

func TestAttendance_LowAttendanceWarning(t *testing.T) {
	f := newSchoolFixture(t)
	ctx := context.Background()
	// 5 school days: 3 absent, 2 present -> 40%
	days := []struct {
		date   string
		status models.AttendanceStatus
	}{
		{"2026-09-28", models.AttendancePresent},
		{"2026-09-29", models.AttendancePresent},
		{"2026-09-30", models.AttendanceAbsent},
		{"2026-10-01", models.AttendanceAbsent},
		{"2026-10-02", models.AttendanceAbsent},
	}
	for _, d := range days {
		if _, err := f.attendance.MarkAttendance(ctx, actorOf(f.admin), &models.MarkAttendanceRequest{
			ClassID: f.classA.ID, Date: d.date,
			Records: []models.AttendanceEntry{{StudentID: f.studentA.ID, Status: d.status}},
		}); err != nil {
			t.Fatalf("mark %s failed: %v", d.date, err)
		}
	}
	if got := f.notifications.countFor(f.parentUser.ID, models.NotificationAttendanceLow); got != 1 {
		t.Fatalf("expected exactly one low-attendance warning for the parent, got %d", got)
	}

	// A further absence while already below the threshold must not repeat the warning
	if _, err := f.attendance.MarkAttendance(ctx, actorOf(f.admin), &models.MarkAttendanceRequest{
		ClassID: f.classA.ID, Date: "2026-10-05",
		Records: []models.AttendanceEntry{{StudentID: f.studentA.ID, Status: models.AttendanceAbsent}},
	}); err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	if got := f.notifications.countFor(f.parentUser.ID, models.NotificationAttendanceLow); got != 1 {
		t.Errorf("warning should not repeat while already below threshold, got %d", got)
	}
}

func TestAttendance_LowAttendanceWarningWhenBackfilling(t *testing.T) {
	f := newSchoolFixture(t)
	ctx := context.Background()
	// Mark newest day first, then fill in earlier days (reverse chronological order)
	for _, d := range []string{"2026-10-06", "2026-10-05", "2026-10-02", "2026-10-01", "2026-09-30"} {
		if _, err := f.attendance.MarkAttendance(ctx, actorOf(f.admin), &models.MarkAttendanceRequest{
			ClassID: f.classA.ID, Date: d,
			Records: []models.AttendanceEntry{{StudentID: f.studentA.ID, Status: models.AttendanceAbsent}},
		}); err != nil {
			t.Fatalf("mark %s failed: %v", d, err)
		}
	}
	if got := f.notifications.countFor(f.studentUserA.ID, models.NotificationAttendanceLow); got != 1 {
		t.Errorf("expected one warning after back-filling, got %d", got)
	}
}

func TestSummarizeAttendance(t *testing.T) {
	recs := []*models.Attendance{
		{Status: models.AttendancePresent},
		{Status: models.AttendancePresent},
		{Status: models.AttendanceLate},
		{Status: models.AttendanceAbsent},
		{Status: models.AttendanceExcused},
	}
	s := SummarizeAttendance(1, recs)
	if s.Total != 5 || s.Present != 2 || s.Late != 1 || s.Absent != 1 || s.Excused != 1 {
		t.Fatalf("unexpected counts: %+v", s)
	}
	// (2 present + 1 late) / (5 total - 1 excused) = 75%
	if s.Percentage != 75 {
		t.Errorf("expected 75%%, got %.2f", s.Percentage)
	}
	if empty := SummarizeAttendance(1, nil); empty.Percentage != 100 {
		t.Errorf("expected 100%% with no records, got %.2f", empty.Percentage)
	}
}

func TestAttendance_StudentReportAccess(t *testing.T) {
	f := newSchoolFixture(t)
	ctx := context.Background()
	cases := []struct {
		name    string
		actor   Actor
		student int64
		allowed bool
	}{
		{"admin", actorOf(f.admin), f.studentA.ID, true},
		{"homeroom teacher", actorOf(f.homeroomTeacher), f.studentA.ID, true},
		{"unrelated teacher", actorOf(f.otherTeacher), f.studentA.ID, false},
		{"student self", actorOf(f.studentUserA), f.studentA.ID, true},
		{"other student", actorOf(f.studentUserB), f.studentA.ID, false},
		{"linked parent", actorOf(f.parentUser), f.studentA.ID, true},
		{"unlinked parent", actorOf(f.parentUser), f.studentB.ID, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := f.attendance.GetStudentReport(ctx, tc.actor, tc.student, "", "")
			if tc.allowed && err != nil {
				t.Errorf("expected access, got %v", err)
			}
			if !tc.allowed && !errors.Is(err, ErrForbidden) {
				t.Errorf("expected ErrForbidden, got %v", err)
			}
		})
	}
}

func TestAttendance_ReportRejectsInvertedRange(t *testing.T) {
	f := newSchoolFixture(t)
	if _, err := f.attendance.GetStudentReport(context.Background(), actorOf(f.admin), f.studentA.ID, "2026-10-05", "2026-10-01"); err == nil {
		t.Error("expected error for from > to")
	}
}

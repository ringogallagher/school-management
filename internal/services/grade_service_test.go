package services

import (
	"context"
	"errors"
	"testing"

	"github.com/mzviagintsev/school-management/internal/models"
)

func TestGrade_RecordAndNotify(t *testing.T) {
	f := newSchoolFixture(t)
	g, err := f.gradeSvc.Record(context.Background(), actorOf(f.homeroomTeacher), &models.RecordGradeRequest{
		StudentID: f.studentA.ID, SubjectID: f.math.ID, Marks: 5, Term: " Q1 ",
	})
	if err != nil {
		t.Fatalf("Record failed: %v", err)
	}
	if g.ClassID != f.classA.ID {
		t.Errorf("expected class taken from student enrolment, got %d", g.ClassID)
	}
	if g.Marks != 5 || g.Term != "Q1" {
		t.Errorf("expected mark 5 and trimmed term Q1, got %.3f %q", g.Marks, g.Term)
	}
	if f.notifications.countFor(f.parentUser.ID, models.NotificationGradePosted) != 1 {
		t.Error("expected parent to be notified of the new grade")
	}
}

func TestGrade_RecordValidation(t *testing.T) {
	f := newSchoolFixture(t)
	admin := actorOf(f.admin)
	cases := []struct {
		name string
		req  models.RecordGradeRequest
	}{
		{"mark above 5", models.RecordGradeRequest{StudentID: f.studentA.ID, SubjectID: f.math.ID, Marks: 6, Term: "Q1"}},
		{"mark below 1", models.RecordGradeRequest{StudentID: f.studentA.ID, SubjectID: f.math.ID, Marks: 0, Term: "Q1"}},
		{"negative mark", models.RecordGradeRequest{StudentID: f.studentA.ID, SubjectID: f.math.ID, Marks: -1, Term: "Q1"}},
		{"fractional mark", models.RecordGradeRequest{StudentID: f.studentA.ID, SubjectID: f.math.ID, Marks: 4.5, Term: "Q1"}},
		{"old 100-point mark", models.RecordGradeRequest{StudentID: f.studentA.ID, SubjectID: f.math.ID, Marks: 87, Term: "Q1"}},
		{"missing term", models.RecordGradeRequest{StudentID: f.studentA.ID, SubjectID: f.math.ID, Marks: 4}},
		{"unknown subject", models.RecordGradeRequest{StudentID: f.studentA.ID, SubjectID: 999, Marks: 4, Term: "Q1"}},
		{"unknown student", models.RecordGradeRequest{StudentID: 999, SubjectID: f.math.ID, Marks: 4, Term: "Q1"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := tc.req
			if _, err := f.gradeSvc.Record(context.Background(), admin, &req); err == nil {
				t.Error("expected an error, got nil")
			}
		})
	}
}

func TestGrade_TeacherCannotGradeOtherClass(t *testing.T) {
	f := newSchoolFixture(t)
	_, err := f.gradeSvc.Record(context.Background(), actorOf(f.otherTeacher), &models.RecordGradeRequest{
		StudentID: f.studentA.ID, SubjectID: f.math.ID, Marks: 4, Term: "Q1",
	})
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestGrade_OnlyRecorderOrAdminCanEdit(t *testing.T) {
	f := newSchoolFixture(t)
	ctx := context.Background()
	g, err := f.gradeSvc.Record(ctx, actorOf(f.homeroomTeacher), &models.RecordGradeRequest{
		StudentID: f.studentA.ID, SubjectID: f.math.ID, Marks: 4, Term: "Q1",
	})
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	two := 2.0
	if _, err := f.gradeSvc.Update(ctx, actorOf(f.otherTeacher), g.ID, &models.UpdateGradeRequest{Marks: &two}); !errors.Is(err, ErrForbidden) {
		t.Errorf("other teacher: expected ErrForbidden, got %v", err)
	}
	zero := 0.0
	if _, err := f.gradeSvc.Update(ctx, actorOf(f.homeroomTeacher), g.ID, &models.UpdateGradeRequest{Marks: &zero}); err == nil {
		t.Error("expected 0 to be rejected on the five-point scale")
	}
	updated, err := f.gradeSvc.Update(ctx, actorOf(f.homeroomTeacher), g.ID, &models.UpdateGradeRequest{Marks: &two})
	if err != nil {
		t.Fatalf("recorder update failed: %v", err)
	}
	if updated.Marks != 2 {
		t.Errorf("expected mark updated to 2, got %.2f", updated.Marks)
	}
	if err := f.gradeSvc.Delete(ctx, actorOf(f.admin), g.ID); err != nil {
		t.Errorf("admin delete failed: %v", err)
	}
	if err := f.gradeSvc.Delete(ctx, actorOf(f.admin), g.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound for deleted grade, got %v", err)
	}
}

func TestFinalMark(t *testing.T) {
	cases := map[float64]int{0: 0, 1: 1, 1.49: 1, 1.5: 2, 2.4: 2, 3.5: 4, 4.49: 4, 4.5: 5, 5: 5}
	for avg, want := range cases {
		if got := FinalMark(avg); got != want {
			t.Errorf("FinalMark(%.2f) = %d, want %d", avg, got, want)
		}
	}
}

func TestBuildGradeReport_FivePointAverages(t *testing.T) {
	grades := []*models.GradeWithDetails{
		// Math: 5 and 4 -> average 4.5 -> final 5
		{Grade: models.Grade{SubjectID: 1, Marks: 5}, SubjectName: "Math"},
		{Grade: models.Grade{SubjectID: 1, Marks: 4}, SubjectName: "Math"},
		// Art: a single 3
		{Grade: models.Grade{SubjectID: 2, Marks: 3}, SubjectName: "Art"},
	}
	r := BuildGradeReport(7, "", grades)

	if len(r.Subjects) != 2 || r.Subjects[0].SubjectName != "Art" {
		t.Fatalf("expected 2 subjects sorted by name, got %+v", r.Subjects)
	}
	if m := r.Subjects[1]; m.Average != 4.5 || m.Final != 5 || m.Count != 2 {
		t.Errorf("unexpected Math average: %+v", m)
	}
	if a := r.Subjects[0]; a.Average != 3 || a.Final != 3 {
		t.Errorf("unexpected Art average: %+v", a)
	}
	// (5 + 4 + 3) / 3 = 4
	if r.OverallAverage != 4 {
		t.Errorf("expected overall average 4, got %.2f", r.OverallAverage)
	}

	empty := BuildGradeReport(7, "Q1", nil)
	if empty.OverallAverage != 0 || len(empty.Grades) != 0 || empty.Grades == nil {
		t.Errorf("expected empty, non-nil report, got %+v", empty)
	}
}

func TestGrade_ParentSeesChildReportOnly(t *testing.T) {
	f := newSchoolFixture(t)
	ctx := context.Background()
	if _, err := f.gradeSvc.GetStudentReport(ctx, actorOf(f.parentUser), f.studentA.ID, ""); err != nil {
		t.Errorf("parent should see own child's grades: %v", err)
	}
	if _, err := f.gradeSvc.GetStudentReport(ctx, actorOf(f.parentUser), f.studentB.ID, ""); !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden for another child, got %v", err)
	}
}

func TestGrade_MyReportForStudent(t *testing.T) {
	f := newSchoolFixture(t)
	ctx := context.Background()
	_, _ = f.gradeSvc.Record(ctx, actorOf(f.admin), &models.RecordGradeRequest{
		StudentID: f.studentA.ID, SubjectID: f.math.ID, Marks: 4, Term: "Q1",
	})
	r, err := f.gradeSvc.GetMyReport(ctx, actorOf(f.studentUserA), "Q1")
	if err != nil {
		t.Fatalf("GetMyReport failed: %v", err)
	}
	if len(r.Grades) != 1 || r.OverallAverage != 4 {
		t.Errorf("expected one grade with average 4, got %+v", r)
	}
}

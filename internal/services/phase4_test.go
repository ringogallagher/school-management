package services

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mzviagintsev/school-management/internal/models"
)

type fakeMailer struct {
	mu   sync.Mutex
	sent []string
}

func (f *fakeMailer) Send(ctx context.Context, to, subject, body string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, to+"|"+subject)
	return nil
}

func TestNotification_EmailCopiesToStudentAndParent(t *testing.T) {
	f := newSchoolFixture(t)
	m := &fakeMailer{}
	notifier := NewNotificationService(f.notifications, f.parents)
	notifier.EnableEmail(m, f.users)
	notifier.syncEmail = true

	notifier.NotifyStudentAndParents(context.Background(), f.studentA, models.NotificationGradePosted, "New grade", "Math: 90")
	if len(m.sent) != 2 {
		t.Fatalf("expected 2 e-mails (student + parent), got %v", m.sent)
	}
	joined := strings.Join(m.sent, ",")
	if !strings.Contains(joined, f.studentUserA.Email) || !strings.Contains(joined, f.parentUser.Email) {
		t.Errorf("unexpected recipients: %v", m.sent)
	}
}

func TestReminder_RunOnceNotifiesHomeroomTeacher(t *testing.T) {
	f := newSchoolFixture(t)
	notifier := NewNotificationService(f.notifications, f.parents)
	// Tuesday 2026-10-06; class A has a lesson on Tuesdays (day 2)
	f.attendanceRp.reminderDay = 2
	f.attendanceRp.reminderClasses = []*models.ReminderTarget{{ClassID: f.classA.ID, ClassName: "9A", TeacherUserID: f.homeroomTeacher.ID}}

	rs := NewReminderService(f.attendanceRp, notifier)
	rs.now = func() time.Time { return time.Date(2026, 10, 6, 15, 0, 0, 0, time.UTC) }

	n, err := rs.RunOnce(context.Background())
	if err != nil || n != 1 {
		t.Fatalf("expected 1 reminder, got %d (%v)", n, err)
	}
	if f.notifications.countFor(f.homeroomTeacher.ID, NotificationAttendanceReminder) != 1 {
		t.Error("homeroom teacher should receive a reminder")
	}

	// After attendance is marked, no reminder
	f.attendance.now = rs.now
	if _, err := f.attendance.MarkAttendance(context.Background(), actorOf(f.homeroomTeacher), &models.MarkAttendanceRequest{
		ClassID: f.classA.ID, Records: []models.AttendanceEntry{{StudentID: f.studentA.ID}},
	}); err != nil {
		t.Fatal(err)
	}
	if n, _ := rs.RunOnce(context.Background()); n != 0 {
		t.Errorf("expected no reminders after marking, got %d", n)
	}
}

func TestNextRun(t *testing.T) {
	loc := time.UTC
	if got := nextRun(time.Date(2026, 10, 6, 14, 0, 0, 0, loc), 15); !got.Equal(time.Date(2026, 10, 6, 15, 0, 0, 0, loc)) {
		t.Errorf("same day expected, got %v", got)
	}
	if got := nextRun(time.Date(2026, 10, 6, 15, 0, 0, 0, loc), 15); !got.Equal(time.Date(2026, 10, 7, 15, 0, 0, 0, loc)) {
		t.Errorf("next day expected, got %v", got)
	}
}

func TestUserService_AdminSafety(t *testing.T) {
	users := NewMockUserRepository()
	svc := NewUserService(users)
	ctx := context.Background()
	admin := &models.User{Email: "a@x", FirstName: "A", Role: models.RoleAdmin, IsActive: true}
	_ = users.Create(ctx, admin)
	other := &models.User{Email: "b@x", FirstName: "B", Role: models.RoleTeacher, IsActive: true}
	_ = users.Create(ctx, other)
	actor := actorOf(admin)

	off := false
	if _, err := svc.Update(ctx, actor, admin.ID, &models.UpdateUserRequest{IsActive: &off}); err == nil {
		t.Error("admin must not deactivate themselves")
	}
	if err := svc.Delete(ctx, actor, admin.ID); err == nil {
		t.Error("admin must not delete themselves")
	}
	if _, err := svc.Update(ctx, actor, other.ID, &models.UpdateUserRequest{IsActive: &off}); err != nil {
		t.Errorf("deactivating another user failed: %v", err)
	}
	taken := "a@x"
	if _, err := svc.Update(ctx, actor, other.ID, &models.UpdateUserRequest{Email: &taken}); err == nil {
		t.Error("expected duplicate email to be rejected")
	}
	if err := svc.ResetPassword(ctx, other.ID, "short"); err == nil {
		t.Error("expected short password to be rejected")
	}
	if list, _ := svc.List(ctx, models.RoleTeacher, "", 10, 0); len(list) != 1 {
		t.Errorf("expected 1 teacher, got %d", len(list))
	}
}

func TestUserService_EnsureAdmin(t *testing.T) {
	users := NewMockUserRepository()
	svc := NewUserService(users)
	ctx := context.Background()
	if err := svc.EnsureAdmin(ctx, "boss@x", "supersecret"); err != nil {
		t.Fatal(err)
	}
	if err := svc.EnsureAdmin(ctx, "other@x", "supersecret"); err != nil {
		t.Fatal(err)
	}
	counts, _ := users.CountByRole(ctx)
	if counts[models.RoleAdmin] != 1 {
		t.Errorf("expected exactly one bootstrap admin, got %d", counts[models.RoleAdmin])
	}
}

func TestRegister_RejectsInvalidRoleAndShortPassword(t *testing.T) {
	svc := NewAuthService(NewMockUserRepository(), "s")
	ctx := context.Background()
	if _, err := svc.Register(ctx, &models.RegisterRequest{Email: "x@x", Password: "password123", FirstName: "X", Role: "superuser"}); err == nil {
		t.Error("expected invalid role to be rejected")
	}
	if _, err := svc.Register(ctx, &models.RegisterRequest{Email: "x@x", Password: "short", FirstName: "X", Role: models.RoleStudent}); err == nil {
		t.Error("expected short password to be rejected")
	}
}

func TestTimetable_TeacherQualificationEnforced(t *testing.T) {
	f := newSchoolFixture(t)
	ctx := context.Background()
	art := &models.Subject{Name: "Art", Code: "ART", Credits: 1}
	_ = f.subjects.Create(ctx, art)
	subjects := NewSubjectService(f.subjects, f.teachers)
	if _, err := subjects.AssignToTeacher(ctx, f.homeroomProfile.ID, []int64{art.ID, art.ID}); err != nil {
		t.Fatal(err)
	}
	if got, _ := subjects.TeacherSubjects(ctx, f.homeroomProfile.ID); len(got) != 1 {
		t.Errorf("duplicates should be removed, got %d subjects", len(got))
	}
	// homeroom teacher is qualified only for Art -> Math lesson rejected
	if _, err := f.timetableS.Create(ctx, f.lesson(f.classA.ID, f.homeroomProfile.ID, 1, "08:00", "08:45", "")); err == nil ||
		!strings.Contains(err.Error(), "not assigned") {
		t.Errorf("expected qualification error, got %v", err)
	}
	// other teacher has no assignments -> unrestricted
	if _, err := f.timetableS.Create(ctx, f.lesson(f.classA.ID, f.otherProfile.ID, 1, "08:00", "08:45", "")); err != nil {
		t.Errorf("unrestricted teacher should be allowed: %v", err)
	}
}

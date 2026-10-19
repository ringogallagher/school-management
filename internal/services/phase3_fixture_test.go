package services

import (
	"context"
	"testing"
	"time"

	"github.com/mzviagintsev/school-management/internal/models"
)

// schoolFixture wires every Phase 3 service to in-memory mocks and seeds a small school:
//
//	class A (homeroom: homeroomTeacher), class B (homeroom: otherTeacher)
//	studentA in class A, studentB in class B
//	parent linked to studentA
//	subject Math (4 credits)
type schoolFixture struct {
	users         *MockUserRepository
	classes       *MockClassRepository
	teachers      *MockTeacherRepository
	students      *MockStudentRepository
	subjects      *MockSubjectRepository
	parents       *MockParentRepository
	attendanceRp  *MockAttendanceRepository
	grades        *MockGradeRepository
	timetable     *MockTimetableRepository
	notifications *MockNotificationRepository

	attendance *AttendanceService
	gradeSvc   *GradeService
	timetableS *TimetableService

	admin, homeroomTeacher, otherTeacher, parentUser, studentUserA, studentUserB *models.User
	homeroomProfile, otherProfile                                                *models.Teacher
	classA, classB                                                               *models.Class
	studentA, studentB                                                           *models.Student
	math                                                                         *models.Subject
	today                                                                        time.Time
}

func newSchoolFixture(t *testing.T) *schoolFixture {
	t.Helper()
	ctx := context.Background()
	f := &schoolFixture{
		users:         NewMockUserRepository(),
		classes:       NewMockClassRepository(),
		teachers:      NewMockTeacherRepository(),
		students:      NewMockStudentRepository(),
		subjects:      NewMockSubjectRepository(),
		parents:       NewMockParentRepository(),
		attendanceRp:  NewMockAttendanceRepository(),
		timetable:     NewMockTimetableRepository(),
		notifications: NewMockNotificationRepository(),
		today:         time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC),
	}
	f.grades = NewMockGradeRepository(f.subjects)

	mkUser := func(email string, role models.UserRole) *models.User {
		u := &models.User{Email: email, FirstName: email, Role: role, IsActive: true}
		_ = f.users.Create(ctx, u)
		return u
	}
	f.admin = mkUser("admin@x", models.RoleAdmin)
	f.homeroomTeacher = mkUser("t1@x", models.RoleTeacher)
	f.otherTeacher = mkUser("t2@x", models.RoleTeacher)
	f.parentUser = mkUser("p@x", models.RoleParent)
	f.studentUserA = mkUser("sa@x", models.RoleStudent)
	f.studentUserB = mkUser("sb@x", models.RoleStudent)

	f.homeroomProfile = &models.Teacher{UserID: f.homeroomTeacher.ID}
	_ = f.teachers.Create(ctx, f.homeroomProfile)
	f.otherProfile = &models.Teacher{UserID: f.otherTeacher.ID}
	_ = f.teachers.Create(ctx, f.otherProfile)

	f.classA = &models.Class{Name: "9A", Level: "9", TeacherID: f.homeroomTeacher.ID}
	_ = f.classes.Create(ctx, f.classA)
	f.classB = &models.Class{Name: "9B", Level: "9", TeacherID: f.otherTeacher.ID}
	_ = f.classes.Create(ctx, f.classB)

	f.studentA = &models.Student{UserID: f.studentUserA.ID, ClassID: f.classA.ID, StudentNumber: "A1"}
	_ = f.students.Create(ctx, f.studentA)
	f.studentB = &models.Student{UserID: f.studentUserB.ID, ClassID: f.classB.ID, StudentNumber: "B1"}
	_ = f.students.Create(ctx, f.studentB)

	parent := &models.Parent{UserID: f.parentUser.ID}
	_ = f.parents.Create(ctx, parent)
	_ = f.parents.LinkStudent(ctx, parent.ID, f.studentA.ID)

	f.math = &models.Subject{Name: "Math", Code: "MATH", Credits: 4}
	_ = f.subjects.Create(ctx, f.math)

	access := NewAccessControl(f.classes, f.teachers, f.parents, f.timetable)
	notifier := NewNotificationService(f.notifications, f.parents)
	f.attendance = NewAttendanceService(f.attendanceRp, f.students, f.classes, access, notifier)
	f.attendance.now = func() time.Time { return f.today.Add(10 * time.Hour) }
	f.gradeSvc = NewGradeService(f.grades, f.students, f.subjects, access, notifier)
	f.timetableS = NewTimetableService(f.timetable, f.classes, f.subjects, f.teachers, f.students)
	return f
}

func actorOf(u *models.User) Actor { return Actor{UserID: u.ID, Role: u.Role} }

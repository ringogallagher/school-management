package services

import (
	"context"
	"time"

	"github.com/mzviagintsev/school-management/internal/models"
)

// UserRepository defines the user data-access operations the auth,
// teacher, and student services depend on. Defining it here (rather than
// depending on the concrete *repository.UserRepository type) lets tests
// substitute an in-memory mock without touching a real database.
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id int64) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id int64) error
	GetByRole(ctx context.Context, role models.UserRole, limit int, offset int) ([]*models.User, error)
	Search(ctx context.Context, role models.UserRole, q string, limit, offset int) ([]*models.User, error)
	CountByRole(ctx context.Context) (map[models.UserRole]int, error)
}

// ClassRepository defines the class data-access operations the class
// service depends on.
type ClassRepository interface {
	Create(ctx context.Context, class *models.Class) error
	GetByID(ctx context.Context, id int64) (*models.Class, error)
	Update(ctx context.Context, class *models.Class) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, limit, offset int) ([]*models.Class, error)
	GetByTeacherID(ctx context.Context, teacherID int64) ([]*models.Class, error)
}

// TeacherRepository defines the teacher data-access operations the
// teacher service depends on.
type TeacherRepository interface {
	Create(ctx context.Context, teacher *models.Teacher) error
	GetByID(ctx context.Context, id int64) (*models.Teacher, error)
	GetByUserID(ctx context.Context, userID int64) (*models.Teacher, error)
	Update(ctx context.Context, teacher *models.Teacher) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, limit, offset int) ([]*models.TeacherWithUser, error)
	SetSubjects(ctx context.Context, teacherID int64, subjectIDs []int64) error
	ListSubjects(ctx context.Context, teacherID int64) ([]*models.Subject, error)
}

// StudentRepository defines the student data-access operations the
// student service depends on.
type StudentRepository interface {
	Create(ctx context.Context, student *models.Student) error
	GetByID(ctx context.Context, id int64) (*models.Student, error)
	GetByUserID(ctx context.Context, userID int64) (*models.Student, error)
	GetByStudentNumber(ctx context.Context, number string) (*models.Student, error)
	Update(ctx context.Context, student *models.Student) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, limit, offset int) ([]*models.StudentWithUser, error)
	ListByClass(ctx context.Context, classID int64, limit, offset int) ([]*models.StudentWithUser, error)
}

// SubjectRepository defines subject data-access operations.
type SubjectRepository interface {
	Create(ctx context.Context, subject *models.Subject) error
	GetByID(ctx context.Context, id int64) (*models.Subject, error)
	GetByCode(ctx context.Context, code string) (*models.Subject, error)
	Update(ctx context.Context, subject *models.Subject) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, limit, offset int) ([]*models.Subject, error)
}

// ParentRepository defines parent data-access operations, including the
// student <-> parent links used for access control and notifications.
type ParentRepository interface {
	Create(ctx context.Context, parent *models.Parent) error
	GetByID(ctx context.Context, id int64) (*models.Parent, error)
	GetByUserID(ctx context.Context, userID int64) (*models.Parent, error)
	LinkStudent(ctx context.Context, parentID, studentID int64) error
	UnlinkStudent(ctx context.Context, parentID, studentID int64) error
	IsParentOf(ctx context.Context, parentUserID, studentID int64) (bool, error)
	ParentUserIDsForStudent(ctx context.Context, studentID int64) ([]int64, error)
	ListChildren(ctx context.Context, parentUserID int64) ([]*models.StudentWithUser, error)
	List(ctx context.Context, limit, offset int) ([]*models.ParentWithUser, error)
}

// AttendanceRepository defines attendance data-access operations.
type AttendanceRepository interface {
	// UpsertBatch inserts or overwrites (by student, class, date) all records atomically.
	UpsertBatch(ctx context.Context, records []*models.Attendance) error
	ListByClassAndDate(ctx context.Context, classID int64, date time.Time) ([]*models.AttendanceWithStudent, error)
	ListByStudent(ctx context.Context, studentID int64, from, to time.Time) ([]*models.Attendance, error)
	SummaryByClass(ctx context.Context, classID int64, from, to time.Time) ([]*models.AttendanceSummary, error)
	// ClassesMissingAttendance returns classes with a lesson on dayOfWeek but no attendance on date.
	ClassesMissingAttendance(ctx context.Context, dayOfWeek int, date time.Time) ([]*models.ReminderTarget, error)
}

// GradeRepository defines grade data-access operations.
type GradeRepository interface {
	Create(ctx context.Context, grade *models.Grade) error
	GetByID(ctx context.Context, id int64) (*models.Grade, error)
	Update(ctx context.Context, grade *models.Grade) error
	Delete(ctx context.Context, id int64) error
	// ListByStudent returns a student's grades; an empty term means all terms.
	ListByStudent(ctx context.Context, studentID int64, term string) ([]*models.GradeWithDetails, error)
	// ListByClassAndSubject returns grades in a class for one subject; an empty term means all terms.
	ListByClassAndSubject(ctx context.Context, classID, subjectID int64, term string) ([]*models.GradeWithDetails, error)
}

// TimetableRepository defines timetable data-access operations.
type TimetableRepository interface {
	Create(ctx context.Context, entry *models.Timetable) error
	GetByID(ctx context.Context, id int64) (*models.Timetable, error)
	Update(ctx context.Context, entry *models.Timetable) error
	Delete(ctx context.Context, id int64) error
	// ListByDay returns every entry on a given day; used for conflict detection.
	ListByDay(ctx context.Context, dayOfWeek int) ([]*models.Timetable, error)
	ListByClass(ctx context.Context, classID int64) ([]*models.TimetableWithDetails, error)
	ListByTeacher(ctx context.Context, teacherID int64) ([]*models.TimetableWithDetails, error)
	// TeacherTeachesClass reports whether the teacher (teachers.id) has any lesson with the class.
	TeacherTeachesClass(ctx context.Context, teacherID, classID int64) (bool, error)
}

// NotificationRepository defines notification data-access operations.
type NotificationRepository interface {
	CreateBatch(ctx context.Context, notifications []*models.Notification) error
	ListByUser(ctx context.Context, userID int64, unreadOnly bool, limit, offset int) ([]*models.Notification, error)
	UnreadCount(ctx context.Context, userID int64) (int, error)
	MarkRead(ctx context.Context, id, userID int64) error
	MarkAllRead(ctx context.Context, userID int64) (int64, error)
	// Broadcast sends one notification to every active user, or only to users with the given role.
	Broadcast(ctx context.Context, role models.UserRole, notifType, title, message string) (int64, error)
}

// NewsRepository defines school news data-access operations.
type NewsRepository interface {
	Create(ctx context.Context, news *models.News) error
	GetByID(ctx context.Context, id int64) (*models.News, error)
	Update(ctx context.Context, news *models.News) error
	Delete(ctx context.Context, id int64) error
	// List returns news with pinned items first, then newest first.
	List(ctx context.Context, limit, offset int) ([]*models.News, error)
}

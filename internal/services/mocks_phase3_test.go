package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mzviagintsev/school-management/internal/models"
)

// In-memory mocks for the Phase 3 repositories (subjects, parents, attendance,
// grades, timetable, notifications).

// ---------- Subjects ----------

type MockSubjectRepository struct {
	subjects map[int64]*models.Subject
	nextID   int64
}

func NewMockSubjectRepository() *MockSubjectRepository {
	return &MockSubjectRepository{subjects: map[int64]*models.Subject{}, nextID: 1}
}

func (m *MockSubjectRepository) Create(ctx context.Context, s *models.Subject) error {
	s.ID = m.nextID
	m.nextID++
	m.subjects[s.ID] = s
	return nil
}
func (m *MockSubjectRepository) GetByID(ctx context.Context, id int64) (*models.Subject, error) {
	return m.subjects[id], nil
}
func (m *MockSubjectRepository) GetByCode(ctx context.Context, code string) (*models.Subject, error) {
	for _, s := range m.subjects {
		if s.Code == code {
			return s, nil
		}
	}
	return nil, nil
}
func (m *MockSubjectRepository) Update(ctx context.Context, s *models.Subject) error {
	m.subjects[s.ID] = s
	return nil
}
func (m *MockSubjectRepository) Delete(ctx context.Context, id int64) error {
	delete(m.subjects, id)
	return nil
}
func (m *MockSubjectRepository) List(ctx context.Context, limit, offset int) ([]*models.Subject, error) {
	var out []*models.Subject
	for _, s := range m.subjects {
		out = append(out, s)
	}
	return out, nil
}

// ---------- Parents ----------

type MockParentRepository struct {
	parents map[int64]*models.Parent
	links   map[[2]int64]bool // {parentID, studentID}
	nextID  int64
}

func NewMockParentRepository() *MockParentRepository {
	return &MockParentRepository{parents: map[int64]*models.Parent{}, links: map[[2]int64]bool{}, nextID: 1}
}

func (m *MockParentRepository) Create(ctx context.Context, p *models.Parent) error {
	p.ID = m.nextID
	m.nextID++
	m.parents[p.ID] = p
	return nil
}
func (m *MockParentRepository) GetByID(ctx context.Context, id int64) (*models.Parent, error) {
	return m.parents[id], nil
}
func (m *MockParentRepository) GetByUserID(ctx context.Context, userID int64) (*models.Parent, error) {
	for _, p := range m.parents {
		if p.UserID == userID {
			return p, nil
		}
	}
	return nil, nil
}
func (m *MockParentRepository) LinkStudent(ctx context.Context, parentID, studentID int64) error {
	m.links[[2]int64{parentID, studentID}] = true
	return nil
}
func (m *MockParentRepository) UnlinkStudent(ctx context.Context, parentID, studentID int64) error {
	delete(m.links, [2]int64{parentID, studentID})
	return nil
}
func (m *MockParentRepository) IsParentOf(ctx context.Context, parentUserID, studentID int64) (bool, error) {
	p, _ := m.GetByUserID(ctx, parentUserID)
	if p == nil {
		return false, nil
	}
	return m.links[[2]int64{p.ID, studentID}], nil
}
func (m *MockParentRepository) ParentUserIDsForStudent(ctx context.Context, studentID int64) ([]int64, error) {
	var ids []int64
	for k := range m.links {
		if k[1] == studentID {
			ids = append(ids, m.parents[k[0]].UserID)
		}
	}
	return ids, nil
}
func (m *MockParentRepository) ListChildren(ctx context.Context, parentUserID int64) ([]*models.StudentWithUser, error) {
	return nil, nil
}

// ---------- Attendance ----------

type MockAttendanceRepository struct {
	records map[string]*models.Attendance // key: student|class|date
	nextID  int64
	failOn  int64 // student ID whose upsert should fail (simulates a DB error)

	// reminder fixtures: classes with lessons on reminderDay
	reminderClasses []*models.ReminderTarget
	reminderDay     int
}

func NewMockAttendanceRepository() *MockAttendanceRepository {
	return &MockAttendanceRepository{records: map[string]*models.Attendance{}, nextID: 1}
}

func attKey(studentID, classID int64, d time.Time) string {
	return fmt.Sprintf("%d|%d|%s", studentID, classID, d.Format(dateLayout))
}

func (m *MockAttendanceRepository) UpsertBatch(ctx context.Context, records []*models.Attendance) error {
	for _, r := range records {
		if r.StudentID == m.failOn {
			return fmt.Errorf("simulated failure")
		}
	}
	for _, r := range records {
		k := attKey(r.StudentID, r.ClassID, r.Date)
		if existing, ok := m.records[k]; ok {
			r.ID = existing.ID
		} else {
			r.ID = m.nextID
			m.nextID++
		}
		m.records[k] = r
	}
	return nil
}
func (m *MockAttendanceRepository) ListByClassAndDate(ctx context.Context, classID int64, date time.Time) ([]*models.AttendanceWithStudent, error) {
	var out []*models.AttendanceWithStudent
	for _, r := range m.records {
		if r.ClassID == classID && r.Date.Equal(date) {
			out = append(out, &models.AttendanceWithStudent{Attendance: *r})
		}
	}
	return out, nil
}
func (m *MockAttendanceRepository) ListByStudent(ctx context.Context, studentID int64, from, to time.Time) ([]*models.Attendance, error) {
	var out []*models.Attendance
	for _, r := range m.records {
		if r.StudentID == studentID && !r.Date.Before(from) && !r.Date.After(to) {
			out = append(out, r)
		}
	}
	return out, nil
}
func (m *MockAttendanceRepository) SummaryByClass(ctx context.Context, classID int64, from, to time.Time) ([]*models.AttendanceSummary, error) {
	by := map[int64]*models.AttendanceSummary{}
	for _, r := range m.records {
		if r.ClassID != classID || r.Date.Before(from) || r.Date.After(to) {
			continue
		}
		s, ok := by[r.StudentID]
		if !ok {
			s = &models.AttendanceSummary{StudentID: r.StudentID}
			by[r.StudentID] = s
		}
		s.Total++
		switch r.Status {
		case models.AttendancePresent:
			s.Present++
		case models.AttendanceAbsent:
			s.Absent++
		case models.AttendanceLate:
			s.Late++
		case models.AttendanceExcused:
			s.Excused++
		}
	}
	var out []*models.AttendanceSummary
	for _, s := range by {
		out = append(out, s)
	}
	return out, nil
}

// ---------- Grades ----------

type MockGradeRepository struct {
	grades   map[int64]*models.Grade
	subjects *MockSubjectRepository
	nextID   int64
}

func NewMockGradeRepository(subjects *MockSubjectRepository) *MockGradeRepository {
	return &MockGradeRepository{grades: map[int64]*models.Grade{}, subjects: subjects, nextID: 1}
}

func (m *MockGradeRepository) Create(ctx context.Context, g *models.Grade) error {
	g.ID = m.nextID
	m.nextID++
	m.grades[g.ID] = g
	return nil
}
func (m *MockGradeRepository) GetByID(ctx context.Context, id int64) (*models.Grade, error) {
	return m.grades[id], nil
}
func (m *MockGradeRepository) Update(ctx context.Context, g *models.Grade) error {
	m.grades[g.ID] = g
	return nil
}
func (m *MockGradeRepository) Delete(ctx context.Context, id int64) error {
	delete(m.grades, id)
	return nil
}
func (m *MockGradeRepository) details(g *models.Grade) *models.GradeWithDetails {
	d := &models.GradeWithDetails{Grade: *g}
	if s := m.subjects.subjects[g.SubjectID]; s != nil {
		d.SubjectName, d.SubjectCode, d.Credits = s.Name, s.Code, s.Credits
	}
	return d
}
func (m *MockGradeRepository) ListByStudent(ctx context.Context, studentID int64, term string) ([]*models.GradeWithDetails, error) {
	var out []*models.GradeWithDetails
	for _, g := range m.grades {
		if g.StudentID == studentID && (term == "" || g.Term == term) {
			out = append(out, m.details(g))
		}
	}
	return out, nil
}
func (m *MockGradeRepository) ListByClassAndSubject(ctx context.Context, classID, subjectID int64, term string) ([]*models.GradeWithDetails, error) {
	var out []*models.GradeWithDetails
	for _, g := range m.grades {
		if g.ClassID == classID && g.SubjectID == subjectID && (term == "" || g.Term == term) {
			out = append(out, m.details(g))
		}
	}
	return out, nil
}

// ---------- Timetable ----------

type MockTimetableRepository struct {
	entries map[int64]*models.Timetable
	nextID  int64
}

func NewMockTimetableRepository() *MockTimetableRepository {
	return &MockTimetableRepository{entries: map[int64]*models.Timetable{}, nextID: 1}
}

func (m *MockTimetableRepository) Create(ctx context.Context, e *models.Timetable) error {
	e.ID = m.nextID
	m.nextID++
	m.entries[e.ID] = e
	return nil
}
func (m *MockTimetableRepository) GetByID(ctx context.Context, id int64) (*models.Timetable, error) {
	return m.entries[id], nil
}
func (m *MockTimetableRepository) Update(ctx context.Context, e *models.Timetable) error {
	m.entries[e.ID] = e
	return nil
}
func (m *MockTimetableRepository) Delete(ctx context.Context, id int64) error {
	delete(m.entries, id)
	return nil
}
func (m *MockTimetableRepository) ListByDay(ctx context.Context, day int) ([]*models.Timetable, error) {
	var out []*models.Timetable
	for _, e := range m.entries {
		if e.DayOfWeek == day {
			out = append(out, e)
		}
	}
	return out, nil
}
func (m *MockTimetableRepository) list(match func(*models.Timetable) bool) []*models.TimetableWithDetails {
	var out []*models.TimetableWithDetails
	for _, e := range m.entries {
		if match(e) {
			out = append(out, &models.TimetableWithDetails{Timetable: *e})
		}
	}
	return out
}
func (m *MockTimetableRepository) ListByClass(ctx context.Context, classID int64) ([]*models.TimetableWithDetails, error) {
	return m.list(func(e *models.Timetable) bool { return e.ClassID == classID }), nil
}
func (m *MockTimetableRepository) ListByTeacher(ctx context.Context, teacherID int64) ([]*models.TimetableWithDetails, error) {
	return m.list(func(e *models.Timetable) bool { return e.TeacherID == teacherID }), nil
}
func (m *MockTimetableRepository) TeacherTeachesClass(ctx context.Context, teacherID, classID int64) (bool, error) {
	for _, e := range m.entries {
		if e.TeacherID == teacherID && e.ClassID == classID {
			return true, nil
		}
	}
	return false, nil
}

// ---------- Notifications ----------

type MockNotificationRepository struct {
	items  []*models.Notification
	nextID int64
}

func NewMockNotificationRepository() *MockNotificationRepository {
	return &MockNotificationRepository{nextID: 1}
}

func (m *MockNotificationRepository) CreateBatch(ctx context.Context, items []*models.Notification) error {
	for _, n := range items {
		n.ID = m.nextID
		m.nextID++
		m.items = append(m.items, n)
	}
	return nil
}
func (m *MockNotificationRepository) ListByUser(ctx context.Context, userID int64, unreadOnly bool, limit, offset int) ([]*models.Notification, error) {
	var out []*models.Notification
	for _, n := range m.items {
		if n.UserID == userID && (!unreadOnly || !n.IsRead) {
			out = append(out, n)
		}
	}
	return out, nil
}
func (m *MockNotificationRepository) UnreadCount(ctx context.Context, userID int64) (int, error) {
	c := 0
	for _, n := range m.items {
		if n.UserID == userID && !n.IsRead {
			c++
		}
	}
	return c, nil
}
func (m *MockNotificationRepository) MarkRead(ctx context.Context, id, userID int64) error {
	for _, n := range m.items {
		if n.ID == id && n.UserID == userID {
			n.IsRead = true
			return nil
		}
	}
	return fmt.Errorf("notification not found")
}
func (m *MockNotificationRepository) MarkAllRead(ctx context.Context, userID int64) (int64, error) {
	var c int64
	for _, n := range m.items {
		if n.UserID == userID && !n.IsRead {
			n.IsRead = true
			c++
		}
	}
	return c, nil
}
func (m *MockNotificationRepository) Broadcast(ctx context.Context, role models.UserRole, notifType, title, message string) (int64, error) {
	return 0, nil
}

// countFor returns how many notifications of a type a user received
func (m *MockNotificationRepository) countFor(userID int64, notifType string) int {
	c := 0
	for _, n := range m.items {
		if n.UserID == userID && strings.EqualFold(n.Type, notifType) {
			c++
		}
	}
	return c
}

// ---------- extra MockUserRepository methods (user management) ----------

func (m *MockUserRepository) Search(ctx context.Context, role models.UserRole, q string, limit, offset int) ([]*models.User, error) {
	var out []*models.User
	q = strings.ToLower(q)
	for _, u := range m.users {
		if role != "" && u.Role != role {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(u.Email+" "+u.FirstName+" "+u.LastName), q) {
			continue
		}
		out = append(out, u)
	}
	return out, nil
}

func (m *MockUserRepository) CountByRole(ctx context.Context) (map[models.UserRole]int, error) {
	counts := map[models.UserRole]int{}
	for _, u := range m.users {
		counts[u.Role]++
	}
	return counts, nil
}

// ClassesMissingAttendance for the attendance mock: classes with a lesson that day (from the
// linked timetable mock) and no attendance on the date.
func (m *MockAttendanceRepository) ClassesMissingAttendance(ctx context.Context, dayOfWeek int, date time.Time) ([]*models.ReminderTarget, error) {
	var out []*models.ReminderTarget
	for _, c := range m.reminderClasses {
		has := false
		for _, r := range m.records {
			if r.ClassID == c.ClassID && r.Date.Equal(date) {
				has = true
			}
		}
		if !has && m.reminderDay == dayOfWeek {
			out = append(out, c)
		}
	}
	return out, nil
}

func (m *MockParentRepository) List(ctx context.Context, limit, offset int) ([]*models.ParentWithUser, error) {
	var out []*models.ParentWithUser
	for _, p := range m.parents {
		out = append(out, &models.ParentWithUser{Parent: *p})
	}
	return out, nil
}

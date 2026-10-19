package memrepo

import (
	"context"
	"fmt"

	"github.com/mzviagintsev/school-management/internal/models"
)

// This file collects in-memory mock repositories shared by the service test
// files in this package. MockUserRepository lives in auth_service_test.go.

// MockClassRepository is an in-memory mock implementation of ClassRepository
type MockClassRepository struct {
	classes map[int64]*models.Class
	nextID  int64
}

// NewMockClassRepository creates a new mock class repository
func NewMockClassRepository() *MockClassRepository {
	return &MockClassRepository{
		classes: make(map[int64]*models.Class),
		nextID:  1,
	}
}

func (m *MockClassRepository) Create(ctx context.Context, class *models.Class) error {
	class.ID = m.nextID
	m.classes[class.ID] = class
	m.nextID++
	return nil
}

func (m *MockClassRepository) GetByID(ctx context.Context, id int64) (*models.Class, error) {
	if c, ok := m.classes[id]; ok {
		return c, nil
	}
	return nil, nil
}

func (m *MockClassRepository) Update(ctx context.Context, class *models.Class) error {
	if _, ok := m.classes[class.ID]; !ok {
		return fmt.Errorf("class not found")
	}
	m.classes[class.ID] = class
	return nil
}

func (m *MockClassRepository) Delete(ctx context.Context, id int64) error {
	if _, ok := m.classes[id]; !ok {
		return fmt.Errorf("class not found")
	}
	delete(m.classes, id)
	return nil
}

func (m *MockClassRepository) List(ctx context.Context, limit, offset int) ([]*models.Class, error) {
	var result []*models.Class
	for _, c := range m.classes {
		result = append(result, c)
	}
	return result, nil
}

func (m *MockClassRepository) GetByTeacherID(ctx context.Context, teacherID int64) ([]*models.Class, error) {
	var result []*models.Class
	for _, c := range m.classes {
		if c.TeacherID == teacherID {
			result = append(result, c)
		}
	}
	return result, nil
}

// MockTeacherRepository is an in-memory mock implementation of TeacherRepository
type MockTeacherRepository struct {
	teachers map[int64]*models.Teacher
	subjects map[int64][]int64 // teacherID -> subject IDs
	nextID   int64
}

// NewMockTeacherRepository creates a new mock teacher repository
func NewMockTeacherRepository() *MockTeacherRepository {
	return &MockTeacherRepository{
		teachers: make(map[int64]*models.Teacher),
		subjects: make(map[int64][]int64),
		nextID:   1,
	}
}

func (m *MockTeacherRepository) Create(ctx context.Context, teacher *models.Teacher) error {
	teacher.ID = m.nextID
	m.teachers[teacher.ID] = teacher
	m.nextID++
	return nil
}

func (m *MockTeacherRepository) GetByID(ctx context.Context, id int64) (*models.Teacher, error) {
	if t, ok := m.teachers[id]; ok {
		return t, nil
	}
	return nil, nil
}

func (m *MockTeacherRepository) GetByUserID(ctx context.Context, userID int64) (*models.Teacher, error) {
	for _, t := range m.teachers {
		if t.UserID == userID {
			return t, nil
		}
	}
	return nil, nil
}

func (m *MockTeacherRepository) Update(ctx context.Context, teacher *models.Teacher) error {
	if _, ok := m.teachers[teacher.ID]; !ok {
		return fmt.Errorf("teacher not found")
	}
	m.teachers[teacher.ID] = teacher
	return nil
}

func (m *MockTeacherRepository) Delete(ctx context.Context, id int64) error {
	if _, ok := m.teachers[id]; !ok {
		return fmt.Errorf("teacher not found")
	}
	delete(m.teachers, id)
	return nil
}

func (m *MockTeacherRepository) List(ctx context.Context, limit, offset int) ([]*models.TeacherWithUser, error) {
	var result []*models.TeacherWithUser
	for _, t := range m.teachers {
		result = append(result, &models.TeacherWithUser{Teacher: *t})
	}
	return result, nil
}

// MockStudentRepository is an in-memory mock implementation of StudentRepository
type MockStudentRepository struct {
	students map[int64]*models.Student
	nextID   int64
}

// NewMockStudentRepository creates a new mock student repository
func NewMockStudentRepository() *MockStudentRepository {
	return &MockStudentRepository{
		students: make(map[int64]*models.Student),
		nextID:   1,
	}
}

func (m *MockStudentRepository) Create(ctx context.Context, student *models.Student) error {
	student.ID = m.nextID
	m.students[student.ID] = student
	m.nextID++
	return nil
}

func (m *MockStudentRepository) GetByID(ctx context.Context, id int64) (*models.Student, error) {
	if s, ok := m.students[id]; ok {
		return s, nil
	}
	return nil, nil
}

func (m *MockStudentRepository) GetByUserID(ctx context.Context, userID int64) (*models.Student, error) {
	for _, s := range m.students {
		if s.UserID == userID {
			return s, nil
		}
	}
	return nil, nil
}

func (m *MockStudentRepository) GetByStudentNumber(ctx context.Context, number string) (*models.Student, error) {
	for _, s := range m.students {
		if s.StudentNumber == number {
			return s, nil
		}
	}
	return nil, nil
}

func (m *MockStudentRepository) Update(ctx context.Context, student *models.Student) error {
	if _, ok := m.students[student.ID]; !ok {
		return fmt.Errorf("student not found")
	}
	m.students[student.ID] = student
	return nil
}

func (m *MockStudentRepository) Delete(ctx context.Context, id int64) error {
	if _, ok := m.students[id]; !ok {
		return fmt.Errorf("student not found")
	}
	delete(m.students, id)
	return nil
}

func (m *MockStudentRepository) List(ctx context.Context, limit, offset int) ([]*models.StudentWithUser, error) {
	var result []*models.StudentWithUser
	for _, s := range m.students {
		result = append(result, &models.StudentWithUser{Student: *s})
	}
	return result, nil
}

func (m *MockStudentRepository) ListByClass(ctx context.Context, classID int64, limit, offset int) ([]*models.StudentWithUser, error) {
	var result []*models.StudentWithUser
	for _, s := range m.students {
		if s.ClassID == classID {
			result = append(result, &models.StudentWithUser{Student: *s})
		}
	}
	return result, nil
}

func (m *MockTeacherRepository) SetSubjects(ctx context.Context, teacherID int64, subjectIDs []int64) error {
	m.subjects[teacherID] = append([]int64(nil), subjectIDs...)
	return nil
}

func (m *MockTeacherRepository) ListSubjects(ctx context.Context, teacherID int64) ([]*models.Subject, error) {
	var out []*models.Subject
	for _, id := range m.subjects[teacherID] {
		out = append(out, &models.Subject{ID: id})
	}
	return out, nil
}

package services

import (
	"context"
	"testing"

	"github.com/mzviagintsev/school-management/internal/models"
)

func seedClass(t *testing.T, classRepo *MockClassRepository, teacherID int64) *models.Class {
	t.Helper()
	class := &models.Class{Name: "Homeroom 9A", Level: "9A", TeacherID: teacherID, MaxStudents: 30}
	if err := classRepo.Create(context.Background(), class); err != nil {
		t.Fatalf("failed to seed class: %v", err)
	}
	return class
}

func TestStudentService_Create_Success(t *testing.T) {
	userRepo := NewMockUserRepository()
	classRepo := NewMockClassRepository()
	studentRepo := NewMockStudentRepository()
	studentService := NewStudentService(studentRepo, classRepo, userRepo)

	class := seedClass(t, classRepo, 1)

	req := &models.CreateStudentRequest{
		Email:         "student@example.com",
		Password:      "password123",
		FirstName:     "Alice",
		LastName:      "Doe",
		ClassID:       class.ID,
		StudentNumber: "S-001",
	}

	student, err := studentService.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if student.StudentNumber != "S-001" {
		t.Errorf("expected student number S-001, got %s", student.StudentNumber)
	}
	if student.ClassID != class.ID {
		t.Errorf("expected class ID %d, got %d", class.ID, student.ClassID)
	}

	user, _ := userRepo.GetByEmail(context.Background(), req.Email)
	if user == nil || user.Role != models.RoleStudent {
		t.Error("expected a user account with role=student to be created")
	}
}

func TestStudentService_Create_UnknownClass(t *testing.T) {
	userRepo := NewMockUserRepository()
	classRepo := NewMockClassRepository()
	studentRepo := NewMockStudentRepository()
	studentService := NewStudentService(studentRepo, classRepo, userRepo)

	req := &models.CreateStudentRequest{
		Email: "student@example.com", Password: "password123", FirstName: "Alice",
		ClassID: 999, StudentNumber: "S-001",
	}

	if _, err := studentService.Create(context.Background(), req); err == nil {
		t.Error("expected error for unknown class, got nil")
	}
}

func TestStudentService_Create_DuplicateStudentNumber(t *testing.T) {
	userRepo := NewMockUserRepository()
	classRepo := NewMockClassRepository()
	studentRepo := NewMockStudentRepository()
	studentService := NewStudentService(studentRepo, classRepo, userRepo)

	class := seedClass(t, classRepo, 1)
	req := &models.CreateStudentRequest{
		Email: "student1@example.com", Password: "password123", FirstName: "Alice",
		ClassID: class.ID, StudentNumber: "S-001",
	}
	if _, err := studentService.Create(context.Background(), req); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}

	req2 := *req
	req2.Email = "student2@example.com"
	if _, err := studentService.Create(context.Background(), &req2); err == nil {
		t.Error("expected error for duplicate student number, got nil")
	}
}

func TestStudentService_Update_MoveClass(t *testing.T) {
	userRepo := NewMockUserRepository()
	classRepo := NewMockClassRepository()
	studentRepo := NewMockStudentRepository()
	studentService := NewStudentService(studentRepo, classRepo, userRepo)

	classA := seedClass(t, classRepo, 1)
	student, err := studentService.Create(context.Background(), &models.CreateStudentRequest{
		Email: "student@example.com", Password: "password123", FirstName: "Alice",
		ClassID: classA.ID, StudentNumber: "S-001",
	})
	if err != nil {
		t.Fatalf("setup Create failed: %v", err)
	}

	classB := seedClass(t, classRepo, 1)

	updated, err := studentService.Update(context.Background(), student.ID, &models.UpdateStudentRequest{ClassID: classB.ID})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.ClassID != classB.ID {
		t.Errorf("expected class ID %d, got %d", classB.ID, updated.ClassID)
	}
}

func TestStudentService_Update_UnknownClass(t *testing.T) {
	userRepo := NewMockUserRepository()
	classRepo := NewMockClassRepository()
	studentRepo := NewMockStudentRepository()
	studentService := NewStudentService(studentRepo, classRepo, userRepo)

	classA := seedClass(t, classRepo, 1)
	student, err := studentService.Create(context.Background(), &models.CreateStudentRequest{
		Email: "student@example.com", Password: "password123", FirstName: "Alice",
		ClassID: classA.ID, StudentNumber: "S-001",
	})
	if err != nil {
		t.Fatalf("setup Create failed: %v", err)
	}

	if _, err := studentService.Update(context.Background(), student.ID, &models.UpdateStudentRequest{ClassID: 999}); err == nil {
		t.Error("expected error moving student to unknown class, got nil")
	}
}

func TestStudentService_Delete_NotFound(t *testing.T) {
	userRepo := NewMockUserRepository()
	classRepo := NewMockClassRepository()
	studentRepo := NewMockStudentRepository()
	studentService := NewStudentService(studentRepo, classRepo, userRepo)

	if err := studentService.Delete(context.Background(), 999); err == nil {
		t.Error("expected error deleting nonexistent student, got nil")
	}
}

func TestStudentService_ListByClass_UnknownClass(t *testing.T) {
	userRepo := NewMockUserRepository()
	classRepo := NewMockClassRepository()
	studentRepo := NewMockStudentRepository()
	studentService := NewStudentService(studentRepo, classRepo, userRepo)

	if _, err := studentService.ListByClass(context.Background(), 999, 20, 0); err == nil {
		t.Error("expected error listing students for unknown class, got nil")
	}
}

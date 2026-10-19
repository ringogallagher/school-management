package services

import (
	"context"
	"testing"

	"github.com/mzviagintsev/school-management/internal/models"
)

func seedTeacherUser(t *testing.T, userRepo *MockUserRepository, email string) *models.User {
	t.Helper()
	user := &models.User{
		Email:     email,
		FirstName: "Jane",
		LastName:  "Smith",
		Role:      models.RoleTeacher,
		IsActive:  true,
	}
	if err := userRepo.Create(context.Background(), user); err != nil {
		t.Fatalf("failed to seed teacher user: %v", err)
	}
	return user
}

func TestClassService_Create_Success(t *testing.T) {
	userRepo := NewMockUserRepository()
	classRepo := NewMockClassRepository()
	classService := NewClassService(classRepo, userRepo)

	teacher := seedTeacherUser(t, userRepo, "teacher@example.com")

	req := &models.CreateClassRequest{
		Name:      "Homeroom 9A",
		Level:     "9A",
		Section:   "A",
		TeacherID: teacher.ID,
	}

	class, err := classService.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if class.ID == 0 {
		t.Error("expected class to have an ID")
	}
	if class.MaxStudents != 30 {
		t.Errorf("expected default max students of 30, got %d", class.MaxStudents)
	}
}

func TestClassService_Create_InvalidTeacher(t *testing.T) {
	userRepo := NewMockUserRepository()
	classRepo := NewMockClassRepository()
	classService := NewClassService(classRepo, userRepo)

	// Seed a student instead of a teacher - assigning them to teach should be rejected.
	student := &models.User{Email: "student@example.com", FirstName: "Al", Role: models.RoleStudent, IsActive: true}
	if err := userRepo.Create(context.Background(), student); err != nil {
		t.Fatalf("failed to seed student user: %v", err)
	}

	req := &models.CreateClassRequest{
		Name:      "Homeroom 9A",
		Level:     "9A",
		TeacherID: student.ID,
	}

	if _, err := classService.Create(context.Background(), req); err == nil {
		t.Error("expected error assigning non-teacher user to class, got nil")
	}
}

func TestClassService_Create_MissingFields(t *testing.T) {
	userRepo := NewMockUserRepository()
	classRepo := NewMockClassRepository()
	classService := NewClassService(classRepo, userRepo)

	if _, err := classService.Create(context.Background(), &models.CreateClassRequest{}); err == nil {
		t.Error("expected error for missing required fields, got nil")
	}
}

func TestClassService_Update_ReassignTeacher(t *testing.T) {
	userRepo := NewMockUserRepository()
	classRepo := NewMockClassRepository()
	classService := NewClassService(classRepo, userRepo)

	teacher1 := seedTeacherUser(t, userRepo, "teacher1@example.com")
	class, err := classService.Create(context.Background(), &models.CreateClassRequest{
		Name: "Homeroom 9A", Level: "9A", TeacherID: teacher1.ID,
	})
	if err != nil {
		t.Fatalf("setup Create failed: %v", err)
	}

	teacher2 := seedTeacherUser(t, userRepo, "teacher2@example.com")

	updated, err := classService.Update(context.Background(), class.ID, &models.UpdateClassRequest{TeacherID: teacher2.ID})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.TeacherID != teacher2.ID {
		t.Errorf("expected teacher ID %d, got %d", teacher2.ID, updated.TeacherID)
	}
}

func TestClassService_Update_NotFound(t *testing.T) {
	userRepo := NewMockUserRepository()
	classRepo := NewMockClassRepository()
	classService := NewClassService(classRepo, userRepo)

	if _, err := classService.Update(context.Background(), 999, &models.UpdateClassRequest{Name: "X"}); err == nil {
		t.Error("expected error updating nonexistent class, got nil")
	}
}

func TestClassService_Delete_NotFound(t *testing.T) {
	userRepo := NewMockUserRepository()
	classRepo := NewMockClassRepository()
	classService := NewClassService(classRepo, userRepo)

	if err := classService.Delete(context.Background(), 999); err == nil {
		t.Error("expected error deleting nonexistent class, got nil")
	}
}

package services

import (
	"context"
	"testing"

	"github.com/mzviagintsev/school-management/internal/models"
)

func TestTeacherService_Create_Success(t *testing.T) {
	userRepo := NewMockUserRepository()
	teacherRepo := NewMockTeacherRepository()
	teacherService := NewTeacherService(teacherRepo, userRepo)

	req := &models.CreateTeacherRequest{
		Email:      "teacher@example.com",
		Password:   "password123",
		FirstName:  "Jane",
		LastName:   "Smith",
		Department: "Mathematics",
		HireDate:   "2023-09-01",
	}

	teacher, err := teacherService.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if teacher.Email != req.Email {
		t.Errorf("expected email %s, got %s", req.Email, teacher.Email)
	}
	if teacher.UserID == 0 {
		t.Error("expected teacher to be linked to a user account")
	}

	// The underlying user should have been created with the teacher role.
	user, _ := userRepo.GetByEmail(context.Background(), req.Email)
	if user == nil || user.Role != models.RoleTeacher {
		t.Error("expected a user account with role=teacher to be created")
	}
}

func TestTeacherService_Create_DuplicateEmail(t *testing.T) {
	userRepo := NewMockUserRepository()
	teacherRepo := NewMockTeacherRepository()
	teacherService := NewTeacherService(teacherRepo, userRepo)

	req := &models.CreateTeacherRequest{Email: "teacher@example.com", Password: "password123", FirstName: "Jane"}

	if _, err := teacherService.Create(context.Background(), req); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}
	if _, err := teacherService.Create(context.Background(), req); err == nil {
		t.Error("expected error for duplicate email, got nil")
	}
}

func TestTeacherService_Create_InvalidHireDate_RollsBackUser(t *testing.T) {
	userRepo := NewMockUserRepository()
	teacherRepo := NewMockTeacherRepository()
	teacherService := NewTeacherService(teacherRepo, userRepo)

	req := &models.CreateTeacherRequest{
		Email: "teacher@example.com", Password: "password123", FirstName: "Jane", HireDate: "not-a-date",
	}

	if _, err := teacherService.Create(context.Background(), req); err == nil {
		t.Error("expected error for invalid hire date, got nil")
	}

	// No user account should have been created since validation failed before creating it.
	user, _ := userRepo.GetByEmail(context.Background(), req.Email)
	if user != nil {
		t.Error("expected no user account to be created when hire date is invalid")
	}
}

func TestTeacherService_Update(t *testing.T) {
	userRepo := NewMockUserRepository()
	teacherRepo := NewMockTeacherRepository()
	teacherService := NewTeacherService(teacherRepo, userRepo)

	teacher, err := teacherService.Create(context.Background(), &models.CreateTeacherRequest{
		Email: "teacher@example.com", Password: "password123", FirstName: "Jane", Department: "Math",
	})
	if err != nil {
		t.Fatalf("setup Create failed: %v", err)
	}

	updated, err := teacherService.Update(context.Background(), teacher.ID, &models.UpdateTeacherRequest{Department: "Physics"})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Department != "Physics" {
		t.Errorf("expected department Physics, got %s", updated.Department)
	}
}

func TestTeacherService_Delete_DeactivatesUser(t *testing.T) {
	userRepo := NewMockUserRepository()
	teacherRepo := NewMockTeacherRepository()
	teacherService := NewTeacherService(teacherRepo, userRepo)

	teacher, err := teacherService.Create(context.Background(), &models.CreateTeacherRequest{
		Email: "teacher@example.com", Password: "password123", FirstName: "Jane",
	})
	if err != nil {
		t.Fatalf("setup Create failed: %v", err)
	}

	if err := teacherService.Delete(context.Background(), teacher.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if _, err := teacherRepo.GetByID(context.Background(), teacher.ID); err != nil {
		t.Fatalf("unexpected error checking teacher repo: %v", err)
	}
	remaining, _ := teacherRepo.GetByID(context.Background(), teacher.ID)
	if remaining != nil {
		t.Error("expected teacher profile to be removed")
	}
}

func TestTeacherService_Delete_NotFound(t *testing.T) {
	userRepo := NewMockUserRepository()
	teacherRepo := NewMockTeacherRepository()
	teacherService := NewTeacherService(teacherRepo, userRepo)

	if err := teacherService.Delete(context.Background(), 999); err == nil {
		t.Error("expected error deleting nonexistent teacher, got nil")
	}
}

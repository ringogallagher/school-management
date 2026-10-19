package services

import (
	"context"
	"testing"

	"github.com/mzviagintsev/school-management/internal/models"
)

func TestSubject_CreateNormalisesAndRejectsDuplicates(t *testing.T) {
	svc := NewSubjectService(NewMockSubjectRepository(), NewMockTeacherRepository())
	ctx := context.Background()

	s, err := svc.Create(ctx, &models.CreateSubjectRequest{Name: " Physics ", Code: "phy101"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if s.Code != "PHY101" || s.Name != "Physics" || s.Credits != 3 {
		t.Errorf("expected normalised code, trimmed name and default credits, got %+v", s)
	}
	if _, err := svc.Create(ctx, &models.CreateSubjectRequest{Name: "Physics 2", Code: "PHY101"}); err == nil {
		t.Error("expected duplicate code to be rejected")
	}
	if _, err := svc.Create(ctx, &models.CreateSubjectRequest{Name: "X"}); err == nil {
		t.Error("expected missing code to be rejected")
	}
}

func TestNotification_MarkReadOnlyOwn(t *testing.T) {
	repo := NewMockNotificationRepository()
	svc := NewNotificationService(repo, NewMockParentRepository())
	ctx := context.Background()
	_ = repo.CreateBatch(ctx, []*models.Notification{{UserID: 1, Title: "a"}, {UserID: 2, Title: "b"}})

	if err := svc.MarkRead(ctx, 2, 1); err == nil {
		t.Error("user 2 must not be able to mark user 1's notification")
	}
	if err := svc.MarkRead(ctx, 1, 1); err != nil {
		t.Errorf("owner should mark own notification: %v", err)
	}
	if n, _ := svc.UnreadCount(ctx, 1); n != 0 {
		t.Errorf("expected 0 unread, got %d", n)
	}
}

func TestNotification_AnnounceValidation(t *testing.T) {
	svc := NewNotificationService(NewMockNotificationRepository(), NewMockParentRepository())
	ctx := context.Background()
	if _, err := svc.Announce(ctx, &models.AnnouncementRequest{Title: "Hi"}); err == nil {
		t.Error("expected error for missing message")
	}
	if _, err := svc.Announce(ctx, &models.AnnouncementRequest{Title: "Hi", Message: "x", Role: "janitor"}); err == nil {
		t.Error("expected error for invalid role")
	}
	if _, err := svc.Announce(ctx, &models.AnnouncementRequest{Title: "Hi", Message: "x", Role: models.RoleParent}); err != nil {
		t.Errorf("valid announcement failed: %v", err)
	}
}

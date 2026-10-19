package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/mzviagintsev/school-management/internal/models"
	"github.com/mzviagintsev/school-management/internal/services"
)

// NotificationHandler handles the current user's notifications and admin announcements
type NotificationHandler struct {
	notificationService *services.NotificationService
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(s *services.NotificationService) *NotificationHandler {
	return &NotificationHandler{notificationService: s}
}

// List GET /api/notifications?unread=true
func (h *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFromRequest(r)
	limit, offset := parsePagination(r)
	unreadOnly := r.URL.Query().Get("unread") == "true"
	items, err := h.notificationService.ListForUser(r.Context(), actor.UserID, unreadOnly, limit, offset)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "success", items)
}

// UnreadCount GET /api/notifications/unread-count
func (h *NotificationHandler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFromRequest(r)
	n, err := h.notificationService.UnreadCount(r.Context(), actor.UserID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "success", map[string]int{"unread": n})
}

// MarkRead POST /api/notifications/{id}/read
func (h *NotificationHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFromRequest(r)
	id, err := urlID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid notification id")
		return
	}
	if err := h.notificationService.MarkRead(r.Context(), actor.UserID, id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeOK(w, "notification marked as read", nil)
}

// MarkAllRead POST /api/notifications/read-all
func (h *NotificationHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFromRequest(r)
	n, err := h.notificationService.MarkAllRead(r.Context(), actor.UserID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "notifications marked as read", map[string]int64{"updated": n})
}

// Announce POST /api/notifications/announce (admin)
func (h *NotificationHandler) Announce(w http.ResponseWriter, r *http.Request) {
	var req models.AnnouncementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	n, err := h.notificationService.Announce(r.Context(), &req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeCreated(w, "announcement sent", map[string]int64{"recipients": n})
}

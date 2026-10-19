package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/mzviagintsev/school-management/internal/models"
	"github.com/mzviagintsev/school-management/internal/services"
)

// NewsHandler handles school news requests
type NewsHandler struct {
	newsService *services.NewsService
}

// NewNewsHandler creates a new news handler
func NewNewsHandler(s *services.NewsService) *NewsHandler {
	return &NewsHandler{newsService: s}
}

// List GET /api/news (public)
func (h *NewsHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)
	items, err := h.newsService.List(r.Context(), limit, offset)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "success", items)
}

// GetByID GET /api/news/{id} (public)
func (h *NewsHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := urlID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid news id")
		return
	}
	item, err := h.newsService.Get(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "success", item)
}

// Create POST /api/news (admin)
func (h *NewsHandler) Create(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFromRequest(r)
	var req models.NewsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	item, err := h.newsService.Create(r.Context(), actor, &req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeCreated(w, "news published", item)
}

// Update PUT /api/news/{id} (admin)
func (h *NewsHandler) Update(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFromRequest(r)
	id, err := urlID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid news id")
		return
	}
	var req models.NewsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	item, err := h.newsService.Update(r.Context(), actor, id, &req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "news updated", item)
}

// Delete DELETE /api/news/{id} (admin)
func (h *NewsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFromRequest(r)
	id, err := urlID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid news id")
		return
	}
	if err := h.newsService.Delete(r.Context(), actor, id); err != nil {
		writeServiceError(w, err)
		return
	}
	writeOK(w, "news deleted", nil)
}

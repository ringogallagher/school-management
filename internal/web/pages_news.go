package web

import (
	"net/http"
	"strings"

	"github.com/mzviagintsev/school-management/internal/models"
)

// ---------- public home page ----------

// home is the public landing page: the school news feed. Signed-in users see it inside the app menu.
func (h *Handler) home(w http.ResponseWriter, r *http.Request) {
	p := h.newPage(r, "School news", "news")
	p.Plain = true
	items, err := h.svc.News.List(r.Context(), 20, 0)
	if err != nil {
		p.Error = errMessage(err)
	}
	p.Data = items
	h.page(w, "home", p)
}

func (h *Handler) newsItem(w http.ResponseWriter, r *http.Request) {
	p := h.newPage(r, "School news", "news")
	item, err := h.svc.News.Get(r.Context(), pathInt(r, "id"))
	if err != nil {
		p.Error = errMessage(err)
		p.Status = http.StatusNotFound
	} else {
		p.Title = item.Title
		p.Data = item
	}
	h.page(w, "news_item", p)
}

// ---------- admin: manage news ----------

type adminNewsData struct {
	Items []*models.News
	Form  *models.News // the item being created or edited (ID 0 = new)
}

func (h *Handler) renderAdminNews(w http.ResponseWriter, r *http.Request, form *models.News, errMsg string) {
	p := h.newPage(r, "News", "adminnews")
	items, err := h.svc.News.List(r.Context(), 100, 0)
	if err != nil && errMsg == "" {
		errMsg = errMessage(err)
	}
	p.Error = errMsg
	p.Data = adminNewsData{Items: items, Form: form}
	h.page(w, "admin_news", p)
}

func (h *Handler) adminNews(w http.ResponseWriter, r *http.Request) {
	form := &models.News{}
	if id := queryInt(r, "edit"); id > 0 {
		item, err := h.svc.News.Get(r.Context(), id)
		if err != nil {
			h.renderAdminNews(w, r, form, errMessage(err))
			return
		}
		form = item
	}
	h.renderAdminNews(w, r, form, "")
}

// adminNewsSave creates a news item (no id in the path) or updates one
func (h *Handler) adminNewsSave(w http.ResponseWriter, r *http.Request) {
	id := pathInt(r, "id")
	req := &models.NewsRequest{
		Title:  r.FormValue("title"),
		Body:   r.FormValue("body"),
		Pinned: r.FormValue("pinned") == "on",
	}
	var err error
	msg := "News published"
	if id == 0 {
		_, err = h.svc.News.Create(r.Context(), actorOf(r), req)
	} else {
		_, err = h.svc.News.Update(r.Context(), actorOf(r), id, req)
		msg = "News updated"
	}
	if err != nil {
		// keep what the admin typed
		form := &models.News{ID: id, Title: req.Title, Body: req.Body, Pinned: req.Pinned}
		h.renderAdminNews(w, r, form, errMessage(err))
		return
	}
	redirect(w, r, "/admin/news", msg)
}

func (h *Handler) adminNewsDelete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.News.Delete(r.Context(), actorOf(r), pathInt(r, "id")); err != nil {
		h.renderAdminNews(w, r, &models.News{}, errMessage(err))
		return
	}
	redirect(w, r, "/admin/news", "News deleted")
}

// newsAuthor is shown under a headline; falls back to the school when the author account is gone
func newsAuthor(n *models.News) string {
	if name := strings.TrimSpace(n.AuthorName); name != "" {
		return name
	}
	return "School office"
}

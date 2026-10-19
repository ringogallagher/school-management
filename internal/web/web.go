// Package web is the server-rendered HTMX frontend. It talks to the same
// service layer as the JSON API (no HTTP round-trip), so every business rule and
// access check is shared between the two front doors.
//
// Authentication uses the same JWT as the API, stored in an HttpOnly,
// SameSite=Strict cookie. State-changing requests additionally must come from
// this origin (Origin/Referer check), which together with SameSite=Strict
// protects against CSRF without a separate token.
package web

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mzviagintsev/school-management/internal/middleware"
	"github.com/mzviagintsev/school-management/internal/models"
	"github.com/mzviagintsev/school-management/internal/services"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

const sessionCookie = "sms_session"

// Services groups everything the frontend calls
type Services struct {
	Auth          *services.AuthService
	Users         *services.UserService
	Classes       *services.ClassService
	Teachers      *services.TeacherService
	Students      *services.StudentService
	Subjects      *services.SubjectService
	Parents       *services.ParentService
	Attendance    *services.AttendanceService
	Grades        *services.GradeService
	Timetable     *services.TimetableService
	Notifications *services.NotificationService
	Reminders     *services.ReminderService
	News          *services.NewsService
	Access        *services.AccessControl
}

// Handler serves the HTML frontend
type Handler struct {
	svc          Services
	pages        map[string]*template.Template
	secureCookie bool
	loginLimiter *middleware.RateLimiter
}

// New parses all templates. secureCookie should be true when served over HTTPS.
func New(svc Services, secureCookie bool, loginLimiter *middleware.RateLimiter) (*Handler, error) {
	h := &Handler{svc: svc, secureCookie: secureCookie, loginLimiter: loginLimiter, pages: map[string]*template.Template{}}

	base, err := template.New("base").Funcs(funcMap).ParseFS(templateFS, "templates/layout.html")
	if err != nil {
		return nil, fmt.Errorf("parse layout: %w", err)
	}
	files, err := fs.Glob(templateFS, "templates/*.html")
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		name := strings.TrimSuffix(path.Base(f), ".html")
		if name == "layout" {
			continue
		}
		t, err := base.Clone()
		if err != nil {
			return nil, err
		}
		if _, err := t.ParseFS(templateFS, f); err != nil {
			return nil, fmt.Errorf("parse %s: %w", f, err)
		}
		h.pages[name] = t
	}
	return h, nil
}

// ---------- template helpers ----------

var weekdays = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

var funcMap = template.FuncMap{
	"date": func(t time.Time) string { return t.Format("2 Jan 2006") },
	"isoDate": func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		return t.Format("2006-01-02")
	},
	"ago": func(t time.Time) string {
		d := time.Since(t)
		switch {
		case d < time.Minute:
			return "just now"
		case d < time.Hour:
			return fmt.Sprintf("%d min ago", int(d.Minutes()))
		case d < 24*time.Hour:
			return fmt.Sprintf("%d h ago", int(d.Hours()))
		default:
			return t.Format("2 Jan")
		}
	},
	"day": func(d int) string {
		if d < 0 || d > 6 {
			return "?"
		}
		return weekdays[d]
	},
	// num prints at most two decimals without trailing zeros: 70, 69.5, 66.33
	"num": func(f float64) string {
		return strconv.FormatFloat(math.Round(f*100)/100, 'f', -1, 64)
	},
	// band maps a five-point mark or average to a CSS modifier (1..5; 0 = no marks yet).
	// Averages are rounded half up, so 4.5 is coloured like a 5.
	"band": func(f float64) string { return strconv.Itoa(services.FinalMark(f)) },
	"pctBand": func(f float64) string {
		switch {
		case f >= 90:
			return "good"
		case f >= services.LowAttendanceThreshold:
			return "ok"
		default:
			return "low"
		}
	},
	"clamp":    func(f float64) float64 { return math.Max(0, math.Min(100, f)) },
	"fullName": func(first, last string) string { return strings.TrimSpace(first + " " + last) },
	"title": func(s string) string {
		if s == "" {
			return s
		}
		return strings.ToUpper(s[:1]) + s[1:]
	},
	"seq": func(n int) []int {
		out := make([]int, n)
		for i := range out {
			out[i] = i
		}
		return out
	},
	"hasID": func(ids map[int64]bool, id int64) bool { return ids[id] },
	// paragraphs splits plain text into paragraphs; templates escape each one, so user text can never inject HTML
	"paragraphs": paragraphs,
	// scale lists the five-point marks best first, for the mark buttons
	"scale":      func() []int { return []int{5, 4, 3, 2, 1} },
	"newsAuthor": newsAuthor,
	// snippet is the start of a text, cut at a word boundary
	"snippet":   snippet,
	"shortDate": func(t time.Time) string { return t.Format("2 Jan") },
	"longDay":   func(t time.Time) string { return t.Format("Monday, 2 January") },
	"weekday":   func(t time.Time) string { return t.Format("Mon") },
	"monthYear": func(t time.Time) string { return t.Format("January 2006") },
	// initial is the first letter of a name, for the round avatar in the header
	"initial": func(s string) string {
		for _, r := range strings.TrimSpace(s) {
			return strings.ToUpper(string(r))
		}
		return "?"
	},
	// dict builds a map for passing several values to a sub-template
	"dict": func(kv ...any) map[string]any {
		m := map[string]any{}
		for i := 0; i+1 < len(kv); i += 2 {
			if k, ok := kv[i].(string); ok {
				m[k] = kv[i+1]
			}
		}
		return m
	},
	"notifLabel": func(t string) string {
		switch t {
		case models.NotificationGradePosted:
			return "Grade"
		case models.NotificationAbsence:
			return "Absence"
		case models.NotificationAttendanceLow:
			return "Attendance"
		case models.NotificationAnnouncement:
			return "Announcement"
		case services.NotificationAttendanceReminder:
			return "Reminder"
		}
		return "Notice"
	},
}

// ---------- page model ----------

// Page is the data passed to every full-page template
type Page struct {
	Title  string
	Nav    string // key of the active navigation item
	User   *models.User
	Flash  string
	Error  string
	Today  time.Time
	Data   any
	Status int
	Plain  bool // the template builds its own cards instead of sitting inside one big card
}

// IsRole is used by templates to show role-specific navigation
func (p *Page) IsRole(roles ...string) bool {
	if p.User == nil {
		return false
	}
	for _, r := range roles {
		if string(p.User.Role) == r {
			return true
		}
	}
	return false
}

type ctxKey int

const userKey ctxKey = 1

// paragraphs splits text on blank lines (single newlines stay inside a paragraph)
func paragraphs(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	var out []string
	for _, p := range strings.Split(text, "\n\n") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// snippet returns the first paragraph of text, shortened to about max characters at a word boundary
func snippet(text string, max int) string {
	first := text
	if ps := paragraphs(text); len(ps) > 0 {
		first = ps[0]
	}
	first = strings.Join(strings.Fields(first), " ")
	runes := []rune(first)
	if len(runes) <= max {
		return first
	}
	cut := string(runes[:max])
	if i := strings.LastIndex(cut, " "); i > max/2 {
		cut = cut[:i]
	}
	return strings.TrimRight(cut, ".,;:!? ") + "…"
}

func currentUser(r *http.Request) *models.User {
	u, _ := r.Context().Value(userKey).(*models.User)
	return u
}

func actorOf(r *http.Request) services.Actor {
	u := currentUser(r)
	if u == nil {
		return services.Actor{}
	}
	return services.Actor{UserID: u.ID, Role: u.Role}
}

func (h *Handler) newPage(r *http.Request, title, nav string) *Page {
	return &Page{
		Title: title,
		Nav:   nav,
		User:  currentUser(r),
		Flash: r.URL.Query().Get("msg"),
		Today: time.Now(),
	}
}

// render executes a full page (layout + content) or, when block is not empty, only that block.
func (h *Handler) render(w http.ResponseWriter, page string, block string, data any, status int) {
	t, ok := h.pages[page]
	if !ok {
		http.Error(w, "template not found: "+page, http.StatusInternalServerError)
		return
	}
	name := block
	if name == "" {
		name = "layout"
		if strings.HasPrefix(page, "login") {
			name = "bare"
		}
	}
	var buf strings.Builder
	if err := t.ExecuteTemplate(&buf, name, data); err != nil {
		slog.Error("template error", "page", page, "block", block, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	_, _ = w.Write([]byte(buf.String()))
}

// page renders a full page; a non-empty p.Error turns the status into 422 so htmx still swaps it
func (h *Handler) page(w http.ResponseWriter, name string, p *Page) {
	status := p.Status
	if status == 0 && p.Error != "" {
		status = http.StatusUnprocessableEntity
	}
	h.render(w, name, "", p, status)
}

// redirect performs a post/redirect/get, using HX-Location for htmx requests so
// the browser URL and history stay correct.
func redirect(w http.ResponseWriter, r *http.Request, to string, msg string) {
	if msg != "" {
		sep := "?"
		if strings.Contains(to, "?") {
			sep = "&"
		}
		to += sep + "msg=" + url.QueryEscape(msg)
	}
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Location", fmt.Sprintf(`{"path":%q,"target":"body"}`, to))
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, to, http.StatusSeeOther)
}

// fullRedirect forces a full page load (used when switching between the bare login
// layout and the app layout, whose <body> attributes differ)
func fullRedirect(w http.ResponseWriter, r *http.Request, to string) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", to)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, to, http.StatusSeeOther)
}

// errMessage turns service errors into text for the user
func errMessage(err error) string {
	switch {
	case errors.Is(err, services.ErrForbidden):
		return "You don't have access to this."
	case errors.Is(err, services.ErrNotFound):
		return "Not found: " + strings.TrimSuffix(err.Error(), ": "+services.ErrNotFound.Error())
	}
	msg := err.Error()
	if msg == "" {
		return "Something went wrong."
	}
	return strings.ToUpper(msg[:1]) + msg[1:] + "."
}

func formInt(r *http.Request, key string) int64 {
	v, _ := strconv.ParseInt(strings.TrimSpace(r.FormValue(key)), 10, 64)
	return v
}

func queryInt(r *http.Request, key string) int64 {
	v, _ := strconv.ParseInt(r.URL.Query().Get(key), 10, 64)
	return v
}

func pathInt(r *http.Request, key string) int64 {
	v, _ := strconv.ParseInt(chi.URLParam(r, key), 10, 64)
	return v
}

// ---------- middleware ----------

// sameOrigin rejects state-changing requests that do not come from this site (CSRF defence in depth)
func sameOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}
		src := r.Header.Get("Origin")
		if src == "" {
			src = r.Header.Get("Referer")
		}
		u, err := url.Parse(src)
		if src == "" || err != nil || u.Host != r.Host {
			http.Error(w, "cross-site request rejected", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// userFromRequest returns the signed-in, active user for the session cookie, or nil.
// Deactivated or deleted users count as signed out immediately, even if their token has not expired yet.
func (h *Handler) userFromRequest(r *http.Request) *models.User {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return nil
	}
	claims, err := h.svc.Auth.ValidateToken(c.Value)
	if err != nil {
		return nil
	}
	user, err := h.svc.Auth.GetUserByID(r.Context(), claims.UserID)
	if err != nil || !user.IsActive {
		return nil
	}
	return user
}

// optionalLogin loads the user when there is a valid session but lets guests through (public pages)
func (h *Handler) optionalLogin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user := h.userFromRequest(r); user != nil {
			r = r.WithContext(context.WithValue(r.Context(), userKey, user))
		}
		next.ServeHTTP(w, r)
	})
}

// requireLogin redirects guests to the sign-in page
func (h *Handler) requireLogin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user := h.userFromRequest(r); user != nil {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey, user)))
			return
		}
		h.clearSession(w)
		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("HX-Redirect", "/login")
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})
}

// requireRole guards a group of pages
func requireRole(roles ...models.UserRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := currentUser(r)
			for _, role := range roles {
				if u != nil && u.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, "forbidden", http.StatusForbidden)
		})
	}
}

func (h *Handler) setSession(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *Handler) clearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: h.secureCookie, SameSite: http.SameSiteStrictMode,
	})
}

// ---------- routes ----------

// Routes mounts the frontend on r
func (h *Handler) Routes(r chi.Router) {
	static, _ := fs.Sub(staticFS, "static")
	fileServer := http.StripPrefix("/static/", http.FileServer(http.FS(static)))
	r.Get("/static/*", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=3600")
		fileServer.ServeHTTP(w, r)
	})

	r.Group(func(r chi.Router) {
		r.Use(sameOrigin)
		r.Get("/login", h.loginForm)
		r.With(h.loginLimiter.Middleware).Post("/login", h.login)
		r.Post("/logout", h.logout)

		// public pages: guests see the school news, signed-in users get their menu around it
		r.Group(func(r chi.Router) {
			r.Use(h.optionalLogin)
			r.Get("/", h.home)
			r.Get("/news/{id}", h.newsItem)
		})

		r.Group(func(r chi.Router) {
			r.Use(h.requireLogin)
			r.Get("/dashboard", h.dashboard)
			r.Get("/notifications", h.notifications)
			r.Get("/notifications/badge", h.notificationBadge)
			r.Post("/notifications/{id}/read", h.notificationRead)
			r.Post("/notifications/read-all", h.notificationsReadAll)
			r.Get("/account", h.account)
			r.Post("/account/password", h.changePassword)
			r.Get("/timetable", h.myTimetable)

			// student / parent / teacher / admin: report pages, access-checked in the service
			r.Get("/students/{id}/grades", h.studentGrades)
			r.Get("/students/{id}/attendance", h.studentAttendance)
			r.Get("/students/{id}/diary", h.studentDiary)
			r.Get("/students/{id}/grades.csv", h.studentGradesCSV)
			r.Get("/students/{id}/attendance.csv", h.studentAttendanceCSV)
			r.With(requireRole(models.RoleStudent)).Get("/my/grades.csv", h.myGradesCSV)
			r.With(requireRole(models.RoleStudent)).Get("/my/attendance.csv", h.myAttendanceCSV)
			r.With(requireRole(models.RoleStudent)).Get("/my/diary", h.myDiary)
			r.With(requireRole(models.RoleStudent)).Get("/my/grades", h.myGrades)
			r.With(requireRole(models.RoleStudent)).Get("/my/attendance", h.myAttendance)
			r.With(requireRole(models.RoleParent)).Get("/children", h.children)

			// teacher + admin
			r.Group(func(r chi.Router) {
				r.Use(requireRole(models.RoleAdmin, models.RoleTeacher))
				r.Get("/attendance", h.attendanceSheet)
				r.Post("/attendance", h.attendanceSave)
				r.Get("/attendance/stats", h.attendanceStats)
				r.Get("/attendance/stats.csv", h.classAttendanceCSV)
				r.Get("/gradebook.csv", h.gradebookCSV)
				r.Get("/gradebook", h.gradebook)
				r.Post("/gradebook", h.gradeAdd)
				r.Post("/gradebook/{id}", h.gradeUpdate)
				r.Post("/gradebook/{id}/delete", h.gradeDelete)
			})

			// admin
			r.Route("/admin", func(r chi.Router) {
				r.Use(requireRole(models.RoleAdmin))
				r.Get("/users", h.adminUsers)
				r.Post("/users", h.adminUserCreate)
				r.Post("/users/{id}/toggle", h.adminUserToggle)
				r.Post("/users/{id}/password", h.adminUserPassword)
				r.Get("/classes", h.adminClasses)
				r.Post("/classes", h.adminClassCreate)
				r.Post("/classes/{id}/delete", h.adminClassDelete)
				r.Get("/subjects", h.adminSubjects)
				r.Post("/subjects", h.adminSubjectCreate)
				r.Post("/subjects/{id}/delete", h.adminSubjectDelete)
				r.Get("/teachers", h.adminTeachers)
				r.Post("/teachers", h.adminTeacherCreate)
				r.Post("/teachers/{id}/subjects", h.adminTeacherSubjects)
				r.Get("/students", h.adminStudents)
				r.Post("/students", h.adminStudentCreate)
				r.Get("/parents", h.adminParents)
				r.Post("/parents", h.adminParentCreate)
				r.Post("/parents/link", h.adminParentLink)
				r.Get("/timetable", h.adminTimetable)
				r.Post("/timetable", h.adminTimetableCreate)
				r.Post("/timetable/{id}/delete", h.adminTimetableDelete)
				r.Get("/news", h.adminNews)
				r.Post("/news", h.adminNewsSave)
				r.Post("/news/{id}", h.adminNewsSave)
				r.Post("/news/{id}/delete", h.adminNewsDelete)
				r.Get("/announce", h.adminAnnounce)
				r.Post("/announce", h.adminAnnounceSend)
				r.Post("/reminders/run", h.adminRunReminders)
			})
		})
	})
}

package web

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/mzviagintsev/school-management/internal/models"
	"github.com/mzviagintsev/school-management/internal/services"
)

// ---------- login / logout ----------

func (h *Handler) loginForm(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		if _, err := h.svc.Auth.ValidateToken(c.Value); err == nil {
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
		}
	}
	h.page(w, "login", &Page{Title: "Sign in", Flash: r.URL.Query().Get("msg")})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(r.FormValue("email"))
	resp, err := h.svc.Auth.Login(r.Context(), &models.LoginRequest{Email: email, Password: r.FormValue("password")})
	if err != nil {
		h.page(w, "login", &Page{Title: "Sign in", Error: errMessage(err), Data: email})
		return
	}
	h.setSession(w, resp.Token, resp.ExpiresAt)
	fullRedirect(w, r, "/dashboard")
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	h.clearSession(w)
	fullRedirect(w, r, "/login?msg="+url.QueryEscape("You have been signed out"))
}

// ---------- shared lookups ----------

// teachableClasses returns the classes the actor may take attendance and grades for
func (h *Handler) teachableClasses(ctx context.Context, actor services.Actor) []*models.Class {
	all := h.sortedClasses(ctx)
	if actor.Role == models.RoleAdmin {
		return all
	}
	var out []*models.Class
	for _, c := range all {
		if h.svc.Access.CanTeachClass(ctx, actor, c.ID) == nil {
			out = append(out, c)
		}
	}
	return out
}

// sortedClasses lists all classes ordered by name (9A, 9B, 10A ...)
func (h *Handler) sortedClasses(ctx context.Context) []*models.Class {
	all, _ := h.svc.Classes.List(ctx, 100, 0)
	sort.SliceStable(all, func(i, j int) bool { return all[i].Name < all[j].Name })
	return all
}

func (h *Handler) classNames(ctx context.Context) map[int64]string {
	names := map[int64]string{}
	classes, _ := h.svc.Classes.List(ctx, 100, 0)
	for _, c := range classes {
		names[c.ID] = c.Name
	}
	return names
}

func todaysLessons(items []*models.TimetableWithDetails, now time.Time) []*models.TimetableWithDetails {
	var out []*models.TimetableWithDetails
	for _, l := range items {
		if l.DayOfWeek == int(now.Weekday()) {
			out = append(out, l)
		}
	}
	return out
}

// ---------- dashboard ----------

type adminDash struct {
	Counts   map[string]int
	Classes  int
	Subjects int
	Pending  []*models.ReminderTarget
}

type teacherDash struct {
	Lessons []*models.TimetableWithDetails
	Classes []*models.Class
	Pending []*models.ReminderTarget
}

type studentDash struct {
	Student    *models.Student
	Lessons    []*models.TimetableWithDetails
	Grades     *models.StudentGradeReport
	Latest     []*models.GradeWithDetails
	Attendance *models.StudentAttendanceReport
	News       []*models.News
	Recent     []*models.Notification
}

type childCard struct {
	Child      *models.StudentWithUser
	ClassName  string
	Grades     *models.StudentGradeReport
	Latest     []*models.GradeWithDetails
	Attendance *models.StudentAttendanceReport
}

func (h *Handler) dashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor := actorOf(r)
	p := h.newPage(r, "Today", "dashboard")
	p.Plain = true

	switch actor.Role {
	case models.RoleAdmin:
		d := adminDash{}
		counts, _ := h.svc.Users.CountByRole(ctx)
		d.Counts = map[string]int{}
		for role, n := range counts {
			d.Counts[string(role)] = n
		}
		classes, _ := h.svc.Classes.List(ctx, 100, 0)
		subjects, _ := h.svc.Subjects.List(ctx, 100, 0)
		d.Classes, d.Subjects = len(classes), len(subjects)
		d.Pending, _ = h.svc.Reminders.Pending(ctx)
		p.Data = d
		h.page(w, "dashboard_admin", p)

	case models.RoleTeacher:
		d := teacherDash{}
		all, _ := h.svc.Timetable.ListForActor(ctx, actor)
		d.Lessons = todaysLessons(all, p.Today)
		d.Classes = h.teachableClasses(ctx, actor)
		mine := map[int64]bool{}
		for _, c := range d.Classes {
			mine[c.ID] = true
		}
		pending, _ := h.svc.Reminders.Pending(ctx)
		for _, t := range pending {
			if mine[t.ClassID] {
				d.Pending = append(d.Pending, t)
			}
		}
		p.Data = d
		h.page(w, "dashboard_teacher", p)

	case models.RoleStudent:
		d := studentDash{}
		all, _ := h.svc.Timetable.ListForActor(ctx, actor)
		d.Lessons = todaysLessons(all, p.Today)
		d.Grades, _ = h.svc.Grades.GetMyReport(ctx, actor, "")
		if d.Grades != nil {
			d.Latest = latestMarks(d.Grades.Grades, 6)
		}
		d.News, _ = h.svc.News.List(ctx, 3, 0)
		d.Attendance, _ = h.svc.Attendance.GetMyReport(ctx, actor, "", "")
		d.Recent, _ = h.svc.Notifications.ListForUser(ctx, actor.UserID, false, 5, 0)
		p.Data = d
		h.page(w, "dashboard_student", p)

	case models.RoleParent:
		p.Data = h.childCards(ctx, actor)
		h.page(w, "dashboard_parent", p)

	default:
		http.Error(w, "unknown role", http.StatusForbidden)
	}
}

func (h *Handler) childCards(ctx context.Context, actor services.Actor) []childCard {
	kids, _ := h.svc.Parents.ListChildren(ctx, actor.UserID)
	names := h.classNames(ctx)
	cards := make([]childCard, 0, len(kids))
	for _, k := range kids {
		c := childCard{Child: k, ClassName: names[k.ClassID]}
		c.Grades, _ = h.svc.Grades.GetStudentReport(ctx, actor, k.ID, "")
		if c.Grades != nil {
			c.Latest = latestMarks(c.Grades.Grades, 5)
		}
		c.Attendance, _ = h.svc.Attendance.GetStudentReport(ctx, actor, k.ID, "", "")
		cards = append(cards, c)
	}
	return cards
}

func (h *Handler) children(w http.ResponseWriter, r *http.Request) {
	p := h.newPage(r, "My children", "children")
	p.Plain = true
	p.Data = h.childCards(r.Context(), actorOf(r))
	h.page(w, "dashboard_parent", p)
}

// ---------- notifications ----------

func (h *Handler) notifications(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	p := h.newPage(r, "Notifications", "notifications")
	unreadOnly := r.URL.Query().Get("filter") == "unread"
	items, err := h.svc.Notifications.ListForUser(r.Context(), u.ID, unreadOnly, 100, 0)
	if err != nil {
		p.Error = errMessage(err)
	}
	p.Data = map[string]any{"Items": items, "UnreadOnly": unreadOnly}
	h.page(w, "notifications", p)
}

func (h *Handler) notificationBadge(w http.ResponseWriter, r *http.Request) {
	n, _ := h.svc.Notifications.UnreadCount(r.Context(), currentUser(r).ID)
	h.render(w, "notifications", "badge", n, http.StatusOK)
}

func (h *Handler) notificationRead(w http.ResponseWriter, r *http.Request) {
	_ = h.svc.Notifications.MarkRead(r.Context(), currentUser(r).ID, pathInt(r, "id"))
	redirect(w, r, "/notifications", "")
}

func (h *Handler) notificationsReadAll(w http.ResponseWriter, r *http.Request) {
	_, _ = h.svc.Notifications.MarkAllRead(r.Context(), currentUser(r).ID)
	redirect(w, r, "/notifications", "All notifications marked as read")
}

// ---------- account ----------

func (h *Handler) account(w http.ResponseWriter, r *http.Request) {
	h.page(w, "account", h.newPage(r, "Account", "account"))
}

func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	p := h.newPage(r, "Account", "account")
	if r.FormValue("new_password") != r.FormValue("confirm_password") {
		p.Error = "The new passwords don't match."
		h.page(w, "account", p)
		return
	}
	if err := h.svc.Auth.ChangePassword(r.Context(), currentUser(r).ID, r.FormValue("old_password"), r.FormValue("new_password")); err != nil {
		p.Error = errMessage(err)
		h.page(w, "account", p)
		return
	}
	redirect(w, r, "/account", "Password changed")
}

// ---------- timetable (week grid) ----------

// weekGrid groups lessons by weekday (Mon..Sat, Sunday only if used)
type weekGrid struct {
	Days []weekDay
}

type weekDay struct {
	Day     int
	Lessons []*models.TimetableWithDetails
	IsToday bool
}

func buildWeek(items []*models.TimetableWithDetails, now time.Time) weekGrid {
	byDay := map[int][]*models.TimetableWithDetails{}
	for _, l := range items {
		byDay[l.DayOfWeek] = append(byDay[l.DayOfWeek], l)
	}
	order := []int{1, 2, 3, 4, 5}
	if len(byDay[6]) > 0 {
		order = append(order, 6)
	}
	if len(byDay[0]) > 0 {
		order = append(order, 0)
	}
	g := weekGrid{}
	for _, d := range order {
		g.Days = append(g.Days, weekDay{Day: d, Lessons: byDay[d], IsToday: d == int(now.Weekday())})
	}
	return g
}

func (h *Handler) myTimetable(w http.ResponseWriter, r *http.Request) {
	p := h.newPage(r, "Timetable", "timetable")
	items, err := h.svc.Timetable.ListForActor(r.Context(), actorOf(r))
	if err != nil {
		p.Error = errMessage(err)
	}
	p.Data = buildWeek(items, p.Today)
	h.page(w, "timetable", p)
}

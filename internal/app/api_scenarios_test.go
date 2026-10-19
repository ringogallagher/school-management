package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// The scenarios below drive the application purely over HTTP, like a real client. The same
// code runs against in-memory repositories (always) and against PostgreSQL (build tag
// "integration"), so a green run shows the whole stack works: router, middleware,
// handlers, services and, in the second case, the SQL and the migrations.

type client struct {
	t    *testing.T
	base string
	hc   *http.Client
}

func newClient(t *testing.T, base string) *client {
	return &client{t: t, base: base, hc: &http.Client{
		Timeout:       10 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

type reply struct {
	Status int
	Header http.Header
	Body   []byte
}

// JSON decodes the body into a map
func (r reply) JSON(t *testing.T) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(r.Body, &m); err != nil {
		t.Fatalf("response is not JSON (status %d): %s", r.Status, r.Body)
	}
	return m
}

// Data returns the "data" object of an API response
func (r reply) Data(t *testing.T) map[string]any {
	t.Helper()
	d, _ := r.JSON(t)["data"].(map[string]any)
	if d == nil {
		t.Fatalf("no data object in response (status %d): %s", r.Status, r.Body)
	}
	return d
}

func (r reply) ID(t *testing.T) int64 {
	t.Helper()
	f, ok := r.Data(t)["id"].(float64)
	if !ok {
		t.Fatalf("no numeric id in response: %s", r.Body)
	}
	return int64(f)
}

func (c *client) do(method, path, token string, body any, headers map[string]string) reply {
	c.t.Helper()
	var rd io.Reader
	if s, ok := body.(string); ok {
		rd = strings.NewReader(s)
	} else if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.base+path, rd)
	if err != nil {
		c.t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		if _, isForm := body.(string); isForm {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		} else {
			req.Header.Set("Content-Type", "application/json")
		}
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return reply{Status: resp.StatusCode, Header: resp.Header, Body: b}
}

func (c *client) get(path, token string) reply         { return c.do("GET", path, token, nil, nil) }
func (c *client) post(path, token string, b any) reply { return c.do("POST", path, token, b, nil) }
func (c *client) put(path, token string, b any) reply  { return c.do("PUT", path, token, b, nil) }
func (c *client) login(email, password string) string {
	c.t.Helper()
	r := c.post("/api/auth/login", "", map[string]string{"email": email, "password": password})
	if r.Status != http.StatusOK {
		c.t.Fatalf("login %s: status %d: %s", email, r.Status, r.Body)
	}
	tok, _ := r.Data(c.t)["token"].(string)
	if tok == "" {
		c.t.Fatalf("login %s: no token in %s", email, r.Body)
	}
	return tok
}

func want(t *testing.T, what string, got reply, status int) {
	t.Helper()
	if got.Status != status {
		t.Fatalf("%s: expected HTTP %d, got %d: %s", what, status, got.Status, got.Body)
	}
}

func contains(t *testing.T, what, body, sub string) {
	t.Helper()
	if !strings.Contains(body, sub) {
		t.Errorf("%s: expected to find %q", what, sub)
	}
}

// runScenarios runs the end-to-end checks. The data it creates uses a unique suffix, so it
// can run repeatedly against the same database.
func runScenarios(t *testing.T, base, adminEmail, adminPassword string) {
	c := newClient(t, base)
	sfx := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	email := func(who string) string { return who + sfx + "@school.test" }
	const pw = "secret-pass-1"

	t.Run("health and security headers", func(t *testing.T) {
		r := c.get("/health", "")
		want(t, "health", r, 200)
		l := c.get("/login", "")
		want(t, "login page", l, 200)
		for h, v := range map[string]string{"X-Content-Type-Options": "nosniff", "X-Frame-Options": "DENY"} {
			if l.Header.Get(h) != v {
				t.Errorf("header %s = %q, want %q", h, l.Header.Get(h), v)
			}
		}
		if !strings.Contains(l.Header.Get("Content-Security-Policy"), "script-src 'self'") {
			t.Errorf("missing or weak CSP: %q", l.Header.Get("Content-Security-Policy"))
		}
	})

	t.Run("authentication", func(t *testing.T) {
		want(t, "wrong password", c.post("/api/auth/login", "", map[string]string{"email": adminEmail, "password": "nope-nope"}), 401)
		want(t, "no token", c.get("/api/classes", ""), 401)
		want(t, "garbage token", c.get("/api/classes", "not.a.token"), 401)
		want(t, "self registration is closed", c.post("/api/auth/register", "", map[string]string{
			"email": email("evil"), "password": pw, "first_name": "E", "last_name": "V", "role": "admin"}), 401)
	})

	admin := c.login(adminEmail, adminPassword)

	// ---- set up a small school through the admin API
	subject := c.post("/api/subjects", admin, map[string]any{"name": "Math " + sfx, "code": "M" + sfx, "credits": 4})
	want(t, "create subject", subject, 201)
	subjectID := subject.ID(t)

	teacher := c.post("/api/teachers", admin, map[string]any{"email": email("teacher"), "password": pw, "first_name": "Anna", "last_name": "Kovacs", "department": "Sciences"})
	want(t, "create teacher", teacher, 201)
	teacherID := teacher.ID(t)
	teacherUserID := int64(teacher.Data(t)["user_id"].(float64))
	want(t, "assign subject to teacher", c.put(fmt.Sprintf("/api/teachers/%d/subjects", teacherID), admin, map[string]any{"subject_ids": []int64{subjectID}}), 200)

	class := c.post("/api/classes", admin, map[string]any{"name": "9A" + sfx, "level": "9", "section": "A", "teacher_id": teacherUserID, "max_students": 25})
	want(t, "create class", class, 201)
	classID := class.ID(t)

	mkStudent := func(who, number string) (id int64) {
		r := c.post("/api/students", admin, map[string]any{"email": email(who), "password": pw, "first_name": who, "last_name": "Pupil", "class_id": classID, "student_number": number + sfx})
		want(t, "create student "+who, r, 201)
		return r.ID(t)
	}
	studentID := mkStudent("bence", "S1")
	otherStudentID := mkStudent("lili", "S2")

	parent := c.post("/api/parents", admin, map[string]any{"email": email("parent"), "password": pw, "first_name": "Eva", "last_name": "Pupil", "relationship": "mother"})
	want(t, "create parent", parent, 201)
	want(t, "link parent to child", c.post(fmt.Sprintf("/api/parents/%d/students/%d", parent.ID(t), studentID), admin, nil), 200)

	want(t, "timetable entry", c.post("/api/timetable", admin, map[string]any{
		"class_id": classID, "subject_id": subjectID, "teacher_id": teacherID, "day_of_week": 1, "start_time": "08:00", "end_time": "08:45", "room_number": "101" + sfx}), 201)
	// the same teacher cannot be in two places at once
	clash := c.post("/api/timetable", admin, map[string]any{
		"class_id": classID, "subject_id": subjectID, "teacher_id": teacherID, "day_of_week": 1, "start_time": "08:30", "end_time": "09:15", "room_number": "202" + sfx})
	if clash.Status != 400 {
		t.Errorf("overlapping lesson must be rejected with 400, got %d: %s", clash.Status, clash.Body)
	}

	teacherTok := c.login(email("teacher"), pw)
	studentTok := c.login(email("bence"), pw)
	parentTok := c.login(email("parent"), pw)

	t.Run("role based access control", func(t *testing.T) {
		want(t, "student cannot record grades", c.post("/api/grades", studentTok, map[string]any{"student_id": studentID, "subject_id": subjectID, "marks": 5, "term": "Q1"}), 403)
		want(t, "teacher cannot create classes", c.post("/api/classes", teacherTok, map[string]any{"name": "X", "level": "1", "section": "A", "teacher_id": teacherUserID}), 403)
		want(t, "student cannot list users", c.get("/api/users", studentTok), 403)
		want(t, "parent sees own child", c.get(fmt.Sprintf("/api/grades/student/%d", studentID), parentTok), 200)
		want(t, "parent cannot see another child", c.get(fmt.Sprintf("/api/grades/student/%d", otherStudentID), parentTok), 403)
		want(t, "student cannot see a classmate", c.get(fmt.Sprintf("/api/grades/student/%d", otherStudentID), studentTok), 403)
	})

	t.Run("five point grades", func(t *testing.T) {
		for _, bad := range []float64{0, 6, 4.5, 87} {
			r := c.post("/api/grades", teacherTok, map[string]any{"student_id": studentID, "subject_id": subjectID, "marks": bad, "term": "Q1"})
			if r.Status != 400 {
				t.Errorf("mark %v must be rejected with 400, got %d", bad, r.Status)
			}
		}
		for _, ok := range []float64{5, 4} {
			want(t, fmt.Sprintf("mark %v", ok), c.post("/api/grades", teacherTok, map[string]any{"student_id": studentID, "subject_id": subjectID, "marks": ok, "term": "Q1"}), 201)
		}
		r := c.get("/api/grades/me", studentTok)
		want(t, "my grades", r, 200)
		d := r.Data(t)
		if avg, _ := d["overall_average"].(float64); avg != 4.5 {
			t.Errorf("overall average = %v, want 4.5", d["overall_average"])
		}
		if _, old := d["gpa"]; old {
			t.Error("the 4.0-scale GPA must be gone from the report")
		}
		subs, _ := d["subjects"].([]any)
		if len(subs) != 1 || subs[0].(map[string]any)["final"].(float64) != 5 {
			t.Errorf("subject final mark should be 4.5 rounded up to 5, got %v", d["subjects"])
		}
	})

	t.Run("notifications reach the parent", func(t *testing.T) {
		r := c.get("/api/notifications", parentTok)
		want(t, "parent notifications", r, 200)
		contains(t, "grade notification", string(r.Body), "grade")
	})

	t.Run("attendance", func(t *testing.T) {
		recs := []map[string]any{{"student_id": studentID, "status": "present"}, {"student_id": otherStudentID, "status": "absent"}}
		want(t, "mark attendance", c.post("/api/attendance", teacherTok, map[string]any{"class_id": classID, "date": "2026-10-05", "records": recs}), 201)
		want(t, "student cannot mark attendance", c.post("/api/attendance", studentTok, map[string]any{"class_id": classID, "records": recs}), 403)
		r := c.get("/api/attendance/me?from=2026-10-01&to=2026-10-31", studentTok)
		want(t, "my attendance", r, 200)
		if p, _ := r.Data(t)["summary"].(map[string]any)["percentage"].(float64); p != 100 {
			t.Errorf("attendance percentage = %v, want 100", p)
		}
	})

	t.Run("csv export", func(t *testing.T) {
		r := c.get("/api/grades/me/export.csv", studentTok)
		want(t, "export", r, 200)
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "text/csv") {
			t.Errorf("content type = %q", r.Header.Get("Content-Type"))
		}
		if !strings.Contains(r.Header.Get("Content-Disposition"), "attachment") {
			t.Errorf("missing attachment header: %q", r.Header.Get("Content-Disposition"))
		}
		body := string(r.Body)
		if !strings.HasPrefix(body, "\xEF\xBB\xBF") {
			t.Error("CSV should start with a UTF-8 BOM")
		}
		contains(t, "csv content", body, "Math "+sfx)
		want(t, "parent cannot export a stranger", c.get(fmt.Sprintf("/api/grades/student/%d/export.csv", otherStudentID), parentTok), 403)
		want(t, "teacher exports the class", c.get(fmt.Sprintf("/api/grades/class/%d/subject/%d/export.csv", classID, subjectID), teacherTok), 200)
		want(t, "class attendance export", c.get(fmt.Sprintf("/api/attendance/class/%d/stats.csv", classID), teacherTok), 200)
		want(t, "student cannot export class data", c.get(fmt.Sprintf("/api/attendance/class/%d/stats.csv", classID), studentTok), 403)
	})

	t.Run("news", func(t *testing.T) {
		want(t, "news are public", c.get("/api/news", ""), 200)
		want(t, "teacher cannot publish", c.post("/api/news", teacherTok, map[string]any{"title": "x", "body": "y"}), 403)
		want(t, "empty title rejected", c.post("/api/news", admin, map[string]any{"title": " ", "body": "y"}), 400)
		xss := `<script>alert(1)</script>` + sfx
		n := c.post("/api/news", admin, map[string]any{"title": xss, "body": "Line one\n\n<img src=x onerror=alert(2)>", "pinned": true})
		want(t, "publish", n, 201)
		id := n.ID(t)

		home := c.get("/", "")
		want(t, "public home page", home, 200)
		html := string(home.Body)
		contains(t, "escaped title", html, "&lt;script&gt;alert(1)&lt;/script&gt;"+sfx)
		if strings.Contains(html, "<script>alert(1)") {
			t.Error("XSS: the news title was rendered as HTML")
		}
		page := c.get(fmt.Sprintf("/news/%d", id), "")
		want(t, "news page", page, 200)
		if strings.Contains(string(page.Body), "<img src=x") {
			t.Error("XSS: the news body was rendered as HTML")
		}
		want(t, "unknown news", c.get("/news/99999999", ""), 404)
		want(t, "admin deletes", c.do("DELETE", fmt.Sprintf("/api/news/%d", id), admin, nil, nil), 200)
	})

	t.Run("web frontend: sessions, CSRF, access", func(t *testing.T) {
		origin := map[string]string{"Origin": base}
		form := func(email, pass string) string {
			return url.Values{"email": {email}, "password": {pass}}.Encode()
		}
		want(t, "login without origin is a CSRF attempt", c.do("POST", "/login", "", form(adminEmail, adminPassword), nil), 403)
		want(t, "login from another site", c.do("POST", "/login", "", form(adminEmail, adminPassword), map[string]string{"Origin": "http://evil.example"}), 403)
		bad := c.do("POST", "/login", "", form(adminEmail, "wrong-wrong"), origin)
		if bad.Status != 422 {
			t.Errorf("wrong password in the web form: got %d, want 422", bad.Status)
		}

		ok := c.do("POST", "/login", "", form(email("bence"), pw), origin)
		want(t, "web login", ok, 303)
		var cookie string
		for _, sc := range ok.Header.Values("Set-Cookie") {
			if strings.HasPrefix(sc, "sms_session=") {
				cookie = strings.SplitN(sc, ";", 2)[0]
				for _, flag := range []string{"HttpOnly", "SameSite=Strict"} {
					if !strings.Contains(sc, flag) {
						t.Errorf("session cookie lacks %s: %s", flag, sc)
					}
				}
			}
		}
		if cookie == "" {
			t.Fatal("no session cookie set")
		}
		withCookie := map[string]string{"Cookie": cookie}

		g := c.do("GET", "/dashboard", "", nil, nil)
		want(t, "guest dashboard redirects", g, 303)
		if g.Header.Get("Location") != "/login" {
			t.Errorf("redirect to %q, want /login", g.Header.Get("Location"))
		}
		d := c.do("GET", "/dashboard", "", nil, withCookie)
		want(t, "student dashboard", d, 200)
		contains(t, "dashboard", string(d.Body), "Hello, bence")
		want(t, "student diary", c.do("GET", "/my/diary", "", nil, withCookie), 200)
		m := c.do("GET", "/my/grades", "", nil, withCookie)
		want(t, "student marks page", m, 200)
		contains(t, "marks page shows the mark", string(m.Body), `class="mark m5"`)
		csv := c.do("GET", "/my/grades.csv", "", nil, withCookie)
		want(t, "web csv export", csv, 200)
		want(t, "student cannot open admin pages", c.do("GET", "/admin/users", "", nil, withCookie), 403)
		want(t, "student cannot open the gradebook", c.do("GET", "/gradebook", "", nil, withCookie), 403)
		want(t, "logout needs same origin", c.do("POST", "/logout", "", "", nil), 403)
	})
}

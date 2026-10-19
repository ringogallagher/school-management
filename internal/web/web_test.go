package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTemplatesParse(t *testing.T) {
	h, err := New(Services{}, false, nil)
	if err != nil {
		t.Fatalf("templates failed to parse: %v", err)
	}
	for _, name := range []string{"login", "home", "news_item", "diary", "admin_news", "dashboard_admin", "dashboard_student", "dashboard_parent", "student_grades", "attendance", "gradebook", "admin_timetable", "notifications"} {
		if _, ok := h.pages[name]; !ok {
			t.Errorf("page %s missing", name)
		}
	}
}

func TestSameOrigin(t *testing.T) {
	h := sameOrigin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	cases := []struct {
		name, origin, referer string
		want                  int
	}{
		{"same origin", "http://school.test", "", http.StatusOK},
		{"referer fallback", "", "http://school.test/attendance", http.StatusOK},
		{"cross site", "http://evil.test", "", http.StatusForbidden},
		{"no origin info", "", "", http.StatusForbidden},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodPost, "http://school.test/attendance", nil)
		if c.origin != "" {
			req.Header.Set("Origin", c.origin)
		}
		if c.referer != "" {
			req.Header.Set("Referer", c.referer)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != c.want {
			t.Errorf("%s: got %d, want %d", c.name, rec.Code, c.want)
		}
	}
	// GET is always allowed
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://school.test/", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("GET should pass, got %d", rec.Code)
	}
}

func TestAssetURLsAreVersioned(t *testing.T) {
	if len(assetVersion) != 10 {
		t.Fatalf("asset version should be a 10 character hash, got %q", assetVersion)
	}
	b, err := templateFS.ReadFile("templates/layout.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"/static/app.css?v={{assetVersion}}", "/static/htmx.min.js?v={{assetVersion}}"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("layout must reference %s", want)
		}
	}
}

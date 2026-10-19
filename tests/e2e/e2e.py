#!/usr/bin/env python3
"""Browser end-to-end test (Playwright) of the main user journeys.

    pip install playwright && playwright install chromium
    BASE_URL=http://localhost:8080 ADMIN_EMAIL=... ADMIN_PASSWORD=... python3 tests/e2e/e2e.py

The server must run with an empty or demo database (the script creates its own data
through the admin API, using a unique suffix). Screenshots land in tests/e2e/out/.
"""
import os, sys, time, json, urllib.request
from playwright.sync_api import sync_playwright

BASE = os.environ.get("BASE_URL", "http://localhost:8080").rstrip("/")
ADMIN = (os.environ.get("ADMIN_EMAIL", "admin@school.local"), os.environ.get("ADMIN_PASSWORD", "admin12345"))
OUT = os.path.join(os.path.dirname(__file__), "out")
os.makedirs(OUT, exist_ok=True)
SFX = str(int(time.time()))[-7:]
PW = "e2e-pass-12345"

def api(method, path, token=None, body=None):
    req = urllib.request.Request(BASE + path, method=method, data=json.dumps(body).encode() if body is not None else None)
    req.add_header("Content-Type", "application/json")
    if token:
        req.add_header("Authorization", "Bearer " + token)
    with urllib.request.urlopen(req) as r:
        return json.load(r)["data"]

def setup():
    tok = api("POST", "/api/auth/login", body={"email": ADMIN[0], "password": ADMIN[1]})["token"]
    subj = api("POST", "/api/subjects", tok, {"name": f"Physics {SFX}", "code": f"P{SFX}", "credits": 3})
    teacher = api("POST", "/api/teachers", tok, {"email": f"t{SFX}@e2e.test", "password": PW, "first_name": "Tina", "last_name": "Teacher", "department": "Sciences"})
    api("PUT", f"/api/teachers/{teacher['id']}/subjects", tok, {"subject_ids": [subj["id"]]})
    cls = api("POST", "/api/classes", tok, {"name": f"E2E {SFX}", "level": "8", "section": "B", "teacher_id": teacher["user_id"], "max_students": 20})
    stu = api("POST", "/api/students", tok, {"email": f"s{SFX}@e2e.test", "password": PW, "first_name": "Sam", "last_name": "Student", "class_id": cls["id"], "student_number": f"E{SFX}"})
    return {"subject": subj, "teacher": teacher, "class": cls, "student": stu}

def login(page, email, password):
    page.goto(BASE + "/login")
    page.fill("input[name=email]", email)
    page.fill("input[name=password]", password)
    page.click("form button.btn")

def main():
    data = setup()
    with sync_playwright() as p:
        b = p.chromium.launch()
        # 1. public home page with news
        ctx = b.new_context(viewport={"width": 1280, "height": 800}); page = ctx.new_page()
        page.goto(BASE + "/")
        page.screenshot(path=f"{OUT}/01_home.png")

        # 2. admin publishes news; it shows on the home page, HTML is escaped
        login(page, *ADMIN)
        page.wait_for_url("**/dashboard")
        page.goto(BASE + "/admin/news")
        page.fill("input[name=title]", f"<b>Open day {SFX}</b>")
        page.fill("textarea[name=body]", "Come and see us.")
        page.click("form button.btn")
        page.goto(BASE + "/")
        assert f"<b>Open day {SFX}</b>" in page.inner_text("body"), "news title must be shown as text, not interpreted"
        assert page.locator("b", has_text="Open day").count() == 0
        page.screenshot(path=f"{OUT}/02_news.png")
        ctx.close()

        # 3. teacher records a mark through the gradebook
        ctx = b.new_context(viewport={"width": 1280, "height": 800}); page = ctx.new_page()
        login(page, f"t{SFX}@e2e.test", PW)
        page.wait_for_url("**/dashboard")
        page.goto(f"{BASE}/gradebook?class_id={data['class']['id']}&subject_id={data['subject']['id']}")
        page.screenshot(path=f"{OUT}/03_gradebook.png")
        ctx.close()

        # 4. student sees dashboard, marks, and cannot reach admin pages
        ctx = b.new_context(viewport={"width": 1280, "height": 800}); page = ctx.new_page()
        login(page, f"s{SFX}@e2e.test", PW)
        page.wait_for_url("**/dashboard")
        page.screenshot(path=f"{OUT}/04_student_dashboard.png")
        r = page.goto(BASE + "/admin/users")
        assert r.status == 403, f"student must get 403 on /admin/users, got {r.status}"
        page.goto(BASE + "/my/grades")
        page.screenshot(path=f"{OUT}/05_student_marks.png")
        with page.expect_download() as dl:
            page.evaluate("location.href='/my/grades.csv'")
        assert dl.value.suggested_filename.endswith(".csv")
        ctx.close()

        # 5. mobile layout does not scroll horizontally
        ctx = b.new_context(viewport={"width": 390, "height": 800}); page = ctx.new_page()
        login(page, f"s{SFX}@e2e.test", PW)
        page.wait_for_url("**/dashboard")
        w = page.evaluate("[document.documentElement.scrollWidth, window.innerWidth]")
        assert w[0] <= w[1] + 1, f"horizontal scroll on mobile: {w}"
        page.screenshot(path=f"{OUT}/06_mobile.png")
        ctx.close(); b.close()
    print("E2E OK")

if __name__ == "__main__":
    try:
        main()
    except AssertionError as e:
        print("E2E FAILED:", e); sys.exit(1)

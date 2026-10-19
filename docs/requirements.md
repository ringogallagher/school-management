# Requirements specification

## 1. Purpose and scope
A web system for school management: accounts and roles, classes, subjects, timetable, attendance, five-point marks, reports, notifications and school news. Fee/payment tracking is outside the delivered scope (it is optional in the work plan).

## 2. Actors
| Actor | Description |
|---|---|
| Guest | Not signed in. Sees the home page with news and the sign-in form. |
| Student | Reads own diary (timetable, marks, attendance), notifications, exports own reports. |
| Parent | Same as a student, for each linked child. |
| Teacher | Marks attendance and records marks for the classes and subjects they teach, sees class reports. |
| Administrator | Manages users, classes, subjects, timetable, news, announcements; has access to all reports. |

## 3. Functional requirements
| ID | Requirement | Verified by |
|---|---|---|
| FR-1 | Users sign in with e-mail and password; self-registration is closed, accounts are created by an administrator. | `TestAPI_*` "authentication" |
| FR-2 | Roles: admin, teacher, student, parent. Every endpoint and page enforces the role; data access is limited to the user's own scope (teacher: own classes/subjects; student: self; parent: linked children). | "role based access control" |
| FR-3 | Admin can search, edit, deactivate, delete users and reset passwords. | service tests |
| FR-4 | CRUD for classes, subjects, teachers (with the subjects they can teach), students, parents (linked to children). | API scenarios |
| FR-5 | Timetable per class/teacher; the system rejects overlapping lessons of a class, a teacher or a room, and subjects the teacher does not teach. | timetable service tests, scenario |
| FR-6 | Attendance is marked per class and date with statuses present / absent / late / excused; reports with percentage. | "attendance" |
| FR-7 | Marks use the five-point scale (whole numbers 1-5); the subject mark is the average rounded half up. Anything else is rejected. | grade service tests, "five point grades" |
| FR-8 | Students and parents see marks and attendance (diary); teachers see a gradebook per class and subject. | web tests, E2E |
| FR-9 | Notifications for new marks, absences and announcements; optional e-mail copy (SMTP or log). A reminder is created at 15:00 for teachers who have not marked today's attendance. | notification/reminder tests |
| FR-10 | Reports can be exported to CSV: student marks, student attendance, class marks, class attendance statistics. Files open correctly in Excel (UTF-8 BOM); cells cannot carry spreadsheet formulas. | `export_test.go`, "csv export" |
| FR-11 | Public home page with school news; administrators publish, pin, edit and delete news. | "news" |
| FR-12 | Aggregated data is available through database views for reporting. | migration 000005 |

## 4. Non-functional requirements
| ID | Requirement | Verified by |
|---|---|---|
| NFR-1 Security | Passwords hashed with bcrypt; JWT sessions; web cookie is HttpOnly + SameSite=Strict (+Secure in production); CSRF protection by Origin check for state-changing web requests; HTML output escaped; security headers (CSP, X-Frame-Options, nosniff, Referrer-Policy); login rate limit. | security tests, `TestLoginRateLimit`, E2E |
| NFR-2 Data integrity | Foreign keys, CHECK constraints (mark range, time order), unique keys. | migrations in CI |
| NFR-3 Performance | Read endpoints answer in under 200 ms (p95) with 20 concurrent users on a 1 000-student data set. | `cmd/loadtest`, see testing.md |
| NFR-4 Usability | A new user completes the main task of their role without help; layout works from 360 px width. | usability plan, E2E mobile check |
| NFR-5 Maintainability | Layered architecture (handlers, services, repositories); services depend on interfaces; CI runs format check, vet, tests, migrations, build. | CI |
| NFR-6 Recoverability | Daily backup, tested restore. | `scripts/backup.sh`, `scripts/restore.sh` |
| NFR-7 Portability | Runs from a Docker image with PostgreSQL 16. | Dockerfile, compose |

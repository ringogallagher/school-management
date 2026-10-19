# Development Roadmap

This document outlines the development plan for the School Management System.

## ✅ Completed (Phase 1: Foundation)

### Database & Architecture
- [x] Database schema design with proper normalization
- [x] PostgreSQL migration files
- [x] ER diagram in schema documentation
- [x] Support for all core entities (User, Student, Teacher, Class, etc)

### Authentication & Security
- [x] User registration with validation
- [x] Password hashing with bcrypt
- [x] JWT-based authentication
- [x] Role-based access control (RBAC)
- [x] Middleware for auth enforcement

### Core API
- [x] Authentication endpoints (register, login, profile)
- [x] Change password functionality
- [x] Proper error handling & logging
- [x] JSON request/response formatting

### Testing & Documentation
- [x] Unit tests for auth service
- [x] API testing guide with curl examples
- [x] Setup guide (Docker & local)
- [x] Architecture documentation
- [x] API documentation

### Development Infrastructure
- [x] Docker & Docker Compose setup
- [x] Makefile for common tasks
- [x] Go module configuration
- [x] Proper project structure & organization

---

## ✅ Completed (Phase 2: Student & Teacher Management)

### Priority 1: Class Management
- [x] Class CRUD endpoints (create, get, update, delete, list)
- [x] Assign teacher to class (with role validation on create & update)
- [x] User list/search endpoint (`/api/users?role=&q=`) — done in Phase 4
- [x] User update endpoint (name, email, active flag) + admin password reset
- [x] Delete user endpoint (soft delete; admins can't delete/deactivate themselves)

### Priority 2: Student Management
- [x] Create student endpoint (creates user account + student profile together)
- [x] Update student endpoint (reassign class)
- [x] Delete student endpoint (deletes profile, deactivates user account)
- [x] List students by class
- [x] Student list/profile endpoints
- [x] Assign parent to student (Phase 3)

### Priority 3: Teacher Management
- [x] Create teacher endpoint (creates user account + teacher profile together)
- [x] Update teacher endpoint
- [x] List teachers (joined with user info)
- [x] Delete teacher endpoint (deletes profile, deactivates user account)
- [x] Assign subjects to teacher (`PUT /api/teachers/{id}/subjects`; enforced by the timetable)

### Bug fix along the way
- [x] Fixed a dependency-injection bug that made `go test ./...` fail to even
      compile: `AuthService` depended on the concrete `*repository.UserRepository`
      struct instead of an interface, so the existing `MockUserRepository` in
      `auth_service_test.go` couldn't be passed in. Added `internal/services/interfaces.go`
      defining `UserRepository`, `ClassRepository`, `TeacherRepository`, and
      `StudentRepository` interfaces; all services now depend on these instead
      of concrete repository types, so they're fully unit-testable with
      in-memory mocks (see `internal/services/mocks_test.go`).

---

## ✅ Completed (Phase 3: Academic Features)

### Prerequisites added along the way
- [x] Subject CRUD (`/api/subjects`) - grades and the timetable need subjects
- [x] Parent accounts + assign parent to student (`/api/parents`) - needed so parents
      can see their child's data and receive notifications
- [x] Migration `000002_academic_constraints` (time-order CHECK on timetable, report indexes)

### Access control (`internal/services/access.go`)
- [x] Central `AccessControl` used by attendance and grades:
  admin = everything; teacher = classes they are homeroom teacher of **or** have a
  timetable lesson with; student = own records; parent = linked children only
- [x] Services receive an explicit `Actor` (user id + role), so rules are unit-tested
      without HTTP; `ErrForbidden` / `ErrNotFound` map to 403 / 404

### Attendance System
- [x] Mark attendance endpoint (bulk, whole class per day, atomic; re-marking overwrites)
- [x] Get attendance report (per student, date range, defaults to last 30 days)
- [x] Calculate attendance percentage: (present + late) / (total - excused)
- [x] Attendance statistics by class (per-student summary)
- [x] Parent + student notification for each absence
- [x] Low-attendance warning (< 75% over 30 days, min. 5 days recorded) - sent once
      when the threshold is crossed, not on every later absence

### Grade Management
- [x] Record grade endpoint (class taken from the student's enrolment)
- [x] Update / delete grade (admin, or the teacher who recorded it)
- [x] Get student grades (optionally by term)
- [x] Get class grades by subject
- [x] Five-point marks (1-5) and per-subject average + final mark (replaced the earlier credit-weighted GPA in Phase 6)
- [x] Grade report (`/api/grades/student/{id}`, `/api/grades/me`)
- [x] Parent + student notification when a grade is posted

### Timetable
- [x] Create / update / delete timetable entry (admin)
- [x] Get class timetable
- [x] Get teacher timetable
- [x] Get "my" timetable (student -> class, teacher -> own lessons)
- [x] Conflict detection: same class, same teacher or same room overlapping on a day
      (back-to-back lessons allowed; updating an entry ignores itself)

### Notifications
- [x] In-app notification history (`/api/notifications`, unread filter, unread count)
- [x] Mark one / all as read (users can only touch their own)
- [x] Parent notifications for grades and attendance
- [x] System announcements (admin broadcast to everyone or one role)
- [x] E-mail notifications (`internal/mailer`: SMTP via `SMTP_*`, or log-only for demos)
- [x] Teacher reminders: daily job at `REMINDER_HOUR` for classes with lessons but no attendance

### Testing
- [x] ~45 new unit tests with in-memory mocks (`mocks_phase3_test.go`, shared
      `schoolFixture` in `phase3_fixture_test.go`), table-driven access-control tests
- [x] End-to-end run against PostgreSQL 16: both migrations (up, down, up), 60+ API calls
      covering every new endpoint and the 403/400 paths

---

## ✅ Completed (Phase 4: HTMX Frontend)

Server-rendered with Go `html/template` + htmx 2 (vendored, no build step), in `internal/web`.
Templates and CSS are embedded into the binary with `go:embed`. The web layer calls the same
services as the JSON API, so every validation and access rule is shared.

### Pages
- [x] Login page (cookie session = same JWT, HttpOnly + SameSite=Strict)
- [x] Admin dashboard (school figures, classes still missing today's attendance, "remind now")
- [x] Admin pages: classes, students, teachers (+ subject qualifications), parents (+ linking),
      subjects, timetable editor (week grid, clash errors inline), announcements, user accounts
- [x] Teacher panel
  - [x] Mark attendance form (P/A/L/E toggles per student, everyone present by default, notes)
  - [x] Enter grades form (gradebook per class/subject/term, inline edit & delete of own marks)
  - [x] Attendance statistics per class and date range
  - [x] View timetable (week grid, today highlighted) and today's lessons on the dashboard
- [x] Student portal: dashboard (average mark, attendance %, today's lessons), weekly diary, marks, attendance, timetable
- [x] Parent portal: card per child (average mark, attendance), child's diary, marks by term and attendance history
- [x] Notifications page for everyone, unread badge in the navigation (polled every 30 s)
- [x] Account page: change password

### HTMX features used
- [x] `hx-boost` navigation (no full reloads), PRG redirects via `HX-Location`
- [x] Live user search (`hx-trigger="input changed delay:300ms"`), filters that reload on change
- [x] Validation errors returned as 422 and swapped in place (configured `responseHandling`)
- [x] In-place editing of marks (`<details>` popovers, no custom JS)
- [x] Works without JavaScript too (plain forms, `<noscript>` submit buttons)

### Testing
- [x] Playwright walkthrough of all four roles (login, every page, attendance save, grade add/edit,
      timetable conflict, announcements, reminders, access denial, password change, mobile layout)

---

## 🚀 Deployment & Optimization (Phase 5)

### Performance
- [x] Database indexes for every report query (migration 000002)
- [x] Pagination on list endpoints
- [ ] Caching layer (Redis) — not needed at school scale; revisit if load testing shows a need
- [x] Static assets embedded and served with cache headers (htmx is pre-minified)

### Security
- [x] Rate limiting on login (API and web), `429` + `Retry-After`
- [x] Public registration closed (anyone could previously register as admin); bootstrap admin from env
- [x] SQL injection prevention — all queries use placeholders (`$1, $2 …`)
- [x] XSS protection — `html/template` auto-escaping + strict Content-Security-Policy
- [x] CSRF protection — SameSite=Strict session cookie + Origin/Referer check on every POST
- [x] Security headers (CSP, X-Frame-Options, nosniff, Referrer-Policy)
- [x] Input validation in services; minimum password length 8
- [x] Fixed: `UserRepository.Update` never saved `password_hash`, so change-password silently did nothing
- [x] Fixed: `RequireRole` panicked when claims were missing
- [x] Deactivated users are logged out of the web UI immediately
- [ ] CORS — not needed (frontend is same-origin); add only if a separate SPA is built

### Deployment
- [x] CI/CD pipeline (GitHub Actions: gofmt, vet, race tests, migrations up/down/up, seed, smoke checks, Docker build)
- [x] Docker image: non-root user, tzdata, seed binary included, working health check (`wget`)
- [x] Environment configuration (`.env.example` documents every variable)
- [x] Demo data seeder (`cmd/seed`, `make seed` / `make docker-seed`)
- [ ] Database backup strategy (e.g. nightly `pg_dump` cron) — hosting-specific
- [ ] Monitoring / log aggregation — hosting-specific

### Documentation
- [x] API guide with curl examples (`API_TESTING.md`) and role/permission table
- [x] Setup guide updated for admin bootstrap and demo data
- [ ] OpenAPI/Swagger spec
- [ ] Short user manual per role

---

## Timeline Estimate

### Week 1-2: Phase 1 (Foundation) ✅
- Database & auth implementation
- Basic API setup
- Documentation

### Week 3-4: Phase 2 (Student & Teacher Management)
- CRUD endpoints for all main entities
- Service layer implementation
- Unit tests

### Week 5-6: Phase 3 (Academic Features)
- Attendance system
- Grade management
- Timetable

### Week 7-8: Phase 4 (Frontend)
- HTMX integration
- Dashboard creation
- User interfaces

### Week 9-10: Phase 5 (Deployment & Polish)
- Performance optimization
- Security hardening
- Deployment setup
- Final testing

---

## Daily Development Checklist

### Each Day
- [ ] Run tests: `make test`
- [ ] Format code: `make fmt`
- [ ] Check logs: `docker-compose logs`
- [ ] Test endpoints with curl
- [ ] Commit changes with clear message

### Each Feature
- [ ] Write repository methods first (data layer)
- [ ] Implement service logic (business logic)
- [ ] Create handler endpoints (HTTP layer)
- [ ] Write unit tests (minimum 80% coverage)
- [ ] Test with curl/Postman
- [ ] Update documentation
- [ ] Commit to git

---

## Learning Goals During Development

### Go Skills
- [x] Package structure
- [x] Error handling
- [x] Interfaces
- [ ] Goroutines & concurrency
- [ ] Context usage patterns
- [ ] Reflection (for validation)

### Database Skills
- [x] SQL fundamentals
- [x] Normalization
- [x] Indexes
- [ ] Query optimization
- [ ] Transaction handling
- [ ] Window functions

### API Design
- [x] REST principles
- [x] HTTP methods
- [x] Status codes
- [x] JSON formatting
- [ ] Versioning
- [ ] Rate limiting

### Testing
- [x] Unit tests
- [ ] Integration tests
- [x] Table-driven tests
- [x] Mock objects
- [ ] Test coverage

---

## Resource Links

### Go Documentation
- https://golang.org/doc/
- https://pkg.go.dev/
- https://go.dev/tour

### PostgreSQL
- https://www.postgresql.org/docs/
- https://www.postgresqltutorial.com/

### REST API Design
- https://restfulapi.net/
- https://swagger.io/

### JWT Authentication
- https://jwt.io/
- https://golang-jwt.github.io/jwt/

### Testing in Go
- https://golang.org/pkg/testing/
- https://github.com/golang/mock

---

## Deployment Checklist

Before going live:

- [ ] Change JWT_SECRET to strong random value
- [ ] Setup SSL/TLS certificates
- [ ] Enable database backups
- [ ] Setup monitoring & alerting
- [ ] Configure logging aggregation
- [ ] Setup rate limiting
- [ ] Configure CORS properly
- [ ] Load test the application
- [ ] Security audit
- [ ] Prepare rollback plan
- [ ] Document runbooks
- [ ] Train support team

---

## Success Criteria

### MVP (Minimum Viable Product)
- [x] Users can register & login
- [x] Admins can manage classes, teachers, and students
- [ ] Teachers can mark attendance
- [ ] Teachers can enter grades
- [ ] Students can view their grades
- [ ] Parents can view child's performance

### v1.0 (Full Release)
- [ ] All CRUD operations complete
- [ ] HTMX frontend complete
- [ ] Performance optimized
- [ ] Security hardened
- [ ] Comprehensive documentation
- [ ] 80%+ test coverage

### v1.1 (Enhancement)
- [ ] Advanced reporting
- [ ] Analytics dashboard
- [ ] SMS notifications
- [ ] Mobile app API

---

This roadmap is flexible and will be updated as the project progresses.
Last updated: 2024-01-15


## Later phases (summary)

- **Phase 6 (2026-10-13):** five-point marks (migration 000003), school news and public home page (migration 000004), weekly diary and new top-bar interface.
- **Phase 7 (2026-10-13):** CSV export, reporting views (migration 000005), backup/restore scripts, HTTP integration tests on in-memory and PostgreSQL back ends, browser E2E test, load-test tool, requirements/database/testing/operations documents, user manuals, OpenAPI.
- **Phase 8 (2026-10-19):** final verification (all tests with the race detector, migrations up/down/up, PostgreSQL integration tests, load test with 1 016 pupils, backup/restore round trip, browser check of every role), news preview fix, documentation clean-up, thesis text in `docs/thesis/`.

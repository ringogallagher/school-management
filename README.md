# School Management System

A web application for running a school's daily record-keeping — attendance, grades, timetable and
notifications — for four kinds of users: administrators, teachers, students and parents.
Built with Go, PostgreSQL and HTMX as a bachelor's thesis project.

## Features

| Area | What it does |
|---|---|
| **Accounts & roles** | Admin, teacher, student, parent. JWT auth (API) and cookie sessions (web). Admin user management: search, deactivate, reset password. |
| **School structure** | Classes with homeroom teachers, students, parents linked to children, subjects with credits, teacher subject qualifications. |
| **Attendance** | Whole-class sheet per day (present / absent / late / excused + note), reports per student and class, attendance % = (present + late) / (days − excused). |
| **Grades** | Five-point marks (whole numbers 1-5, enforced by the service and a database CHECK). Gradebook per class / subject / term, per-subject average and final mark (average rounded half up), overall average. Teachers can only edit marks they entered. |
| **E-diary** | Weekly diary for students and parents: lessons by day with the marks received next to them. |
| **News** | Public home page (`/`) with school news; administrators publish, pin, edit and delete items. |
| **Export** | CSV export of marks and attendance (student and class). |
| **Timetable** | Weekly timetable per class and teacher, with clash detection for class, teacher and room. |
| **Notifications** | In-app (and optional e-mail) messages for new grades, absences, low attendance (< 75 %), announcements and daily "attendance not marked" reminders. |
| **Security** | Role-based access checked in the service layer, login rate limiting, CSRF and XSS protection, CSP, no public self-registration. |

## Tech stack

- **Go 1.21** — chi router, `html/template`, golang-jwt, bcrypt
- **PostgreSQL 15+** — migrations in `migrations/`
- **HTMX 2** — vendored, no frontend build step; templates and CSS are embedded in the binary
- **Docker / Docker Compose**, GitHub Actions CI

## Quick start (Docker)

```bash
make docker-up        # PostgreSQL + migrations + app on http://localhost:8080
make docker-seed      # optional: demo school with teachers, students, parents, grades, attendance
```

Sign in at http://localhost:8080 with the administrator from `docker-compose.yml`
(`admin@school.test` / `admin12345`). After seeding you can also use:

| Role | Email | Password |
|---|---|---|
| Teacher (homeroom 9A) | anna.kovacs@school.test | teacher123 |
| Student (9A) | bence.toth@school.test | student123 |
| Parent of Bence | eva.toth@school.test | parent123 |

## Local development

```bash
cp .env.example .env               # adjust DATABASE_URL, ADMIN_EMAIL, ADMIN_PASSWORD
make migrate-up                    # needs golang-migrate, or apply migrations/*.up.sql with psql
make seed                          # optional demo data (only on an empty database)
make run
make test                          # unit tests; make test-race for the race detector
```

## Architecture

```
Browser (HTMX pages)          API clients (JSON)
        │                            │
  internal/web               internal/handlers
        └──────────┬─────────────────┘
                   ▼
           internal/services   ← business rules, AccessControl, notifications, reminder job
                   ▼
          internal/repository  ← SQL (database/sql + lib/pq)
                   ▼
              PostgreSQL
```

Both front doors call the same services, so a rule such as "a teacher may only mark attendance for
classes they teach" is written once (`internal/services/access.go`) and applies to the web UI and the
API alike. Services depend on repository *interfaces*, which lets the unit tests run against in-memory mocks.

```
cmd/api/            server entry point (configuration, wiring, routes)
cmd/seed/           demo data loader
internal/models/    entities and request/response types
internal/repository/ PostgreSQL data access
internal/services/  business logic + unit tests
internal/handlers/  JSON API handlers
internal/web/       HTMX frontend: handlers, templates/, static/
internal/middleware/ JWT auth, roles, rate limiting, security headers
internal/mailer/    SMTP and log-only e-mail senders
migrations/         SQL schema
```

## Configuration

All settings are environment variables; see `.env.example` for the full list.

| Variable | Purpose |
|---|---|
| `DATABASE_URL` | PostgreSQL connection string |
| `JWT_SECRET` | Signing key for tokens and web sessions — **change in production** |
| `ADMIN_EMAIL`, `ADMIN_PASSWORD` | First administrator, created on startup if none exists |
| `COOKIE_SECURE` | `true` when served over HTTPS |
| `LOGIN_RATE_LIMIT` | Login attempts per IP per minute (default 10) |
| `REMINDER_HOUR` | Hour of the daily attendance reminder (default 15, `-1` disables) |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | Send notification e-mails |
| `EMAIL_LOG` | `true` writes e-mails to the log instead of sending (demo mode) |

## API

The JSON API lives under `/api`. See **API_TESTING.md** for curl examples of every endpoint and a
role/permission table. Main groups: `/api/auth`, `/api/users`, `/api/classes`, `/api/teachers`,
`/api/students`, `/api/parents`, `/api/subjects`, `/api/attendance`, `/api/grades`,
`/api/timetable`, `/api/notifications`.

## Testing

- **Unit tests** — services, middleware, mailer, web helpers (`go test ./...`), all against mocks.
- **CI** (`.github/workflows/ci.yml`) — gofmt, vet, race tests, migrations up/down/up against PostgreSQL,
  seeding the demo school, smoke checks of the running server, Docker build.

## Further documentation

- `SETUP_GUIDE.md` — detailed setup
- `API_TESTING.md` — API reference by example
- `DEVELOPMENT_ROADMAP.md` — what was built in each phase and what is left
- `PROJECT_STRUCTURE.md` — file-by-file description

# Testing

## 1. Test levels
| Level | Where | Needs | Run |
|---|---|---|---|
| Unit | `internal/**/_test.go` (grades, export, news, access control, validators) | nothing | `go test ./...` |
| Web/handler | `internal/web/*_test.go` (CSRF, sessions, rendering, escaping) | nothing | `go test ./...` |
| **Integration over HTTP, in memory** | `internal/app/api_memory_test.go` | nothing | `go test ./internal/app/` |
| **Integration over HTTP, PostgreSQL** | `internal/app/api_pg_test.go` | migrated database (may already contain data: the test creates its own administrator with a unique e-mail) | `TEST_DATABASE_URL=postgres://... go test -tags integration ./internal/app/` |
| Database | migrations up/down/up and demo seed in CI | PostgreSQL | CI job |
| End to end (browser) | `tests/e2e/e2e.py` (Playwright) | running server | `BASE_URL=http://localhost:8080 python3 tests/e2e/e2e.py` |
| Performance | `cmd/loadtest` | running server | see 3 |
| Usability | 4 | volunteers | - |

The HTTP integration scenarios (`runScenarios`) are written once and run against both back ends. They create a subject, teacher, class, students, parent and timetable through the admin API and check: authentication and closed registration, role rules (including parent/child scoping), five-point validation, notifications, attendance, CSV export, news with XSS payloads, web login with CSRF, cookie flags, access to pages, and the login rate limit (`TestLoginRateLimit`).

## 2. Requirement coverage
See the "Verified by" column in [requirements.md](requirements.md).

## 3. Performance test
```
go run ./cmd/loadtest -url http://localhost:8080 -email ADMIN -password PASS -c 20 -d 15s \
   -paths /health,/,/api/news,/api/classes,/api/subjects,/api/students,/api/users
```
Prints requests, req/s, p50/p95/p99/max and error rate per endpoint; exit code 1 if errors exceed 1 %. Use a database seeded with a realistic size (`go run ./cmd/seed`, then add ~1 000 students), run with 1, 10, 20, 50 workers and record the table below. Target (NFR-3): p95 < 200 ms at 20 workers, no errors. Run it from the same host or a neighbour in the same network and note the machine specs.

Measured run (19 October 2026): PostgreSQL 16, 1 016 students, 12 269 marks, 20 240 attendance rows; server, database and load generator on the same cloud VM (so they compete for CPU, which makes the figures conservative); 10 s per level; admin account; 0 % errors at every level. NFR-3 (p95 < 200 ms at 20 workers) is met: the slowest endpoint at 20 workers had p95 = 54.2 ms.

| Endpoint | Workers | req/s | p50 | p95 | p99 | Errors |
|---|---|---|---|---|---|---|
| `/` (news home page) | 1 | 1 045 | 0.89 ms | 1.33 ms | 3.13 ms | 0 % |
| `/` | 10 | 1 704 | 3.9 ms | 17.0 ms | 31.9 ms | 0 % |
| `/` | 20 | 1 146 | 12.8 ms | 54.2 ms | 81.9 ms | 0 % |
| `/` | 50 | 2 219 | 17.7 ms | 58.3 ms | 82.0 ms | 0 % |
| `/api/news` | 20 | 1 628 | 7.1 ms | 41.5 ms | 67.1 ms | 0 % |
| `/api/classes` | 20 | 2 250 | 3.6 ms | 33.6 ms | 56.6 ms | 0 % |
| `/api/students` | 20 | 990 | 16.4 ms | 53.8 ms | 84.3 ms | 0 % |
| `/api/users` | 20 | 1 334 | 10.9 ms | 46.2 ms | 72.3 ms | 0 % |
| `/api/students` | 50 | 1 400 | 32.7 ms | 72.2 ms | 96.2 ms | 0 % |

The numbers come from a shared sandbox VM and a single run per level; repeat them on the target machine before quoting them as a capacity statement.

Login (bcrypt) is deliberately slow and rate limited, so it is not part of the load run.

## 4. Usability test plan
**Participants:** 5 people per role group (teacher, student/parent, administrator), none of whom saw the system before.
**Method:** think-aloud, 15 minutes per person, a facilitator notes time, errors and quotes.
**Tasks**
| Role | Task |
|---|---|
| Student/parent | Find the mark in Physics; find how many lessons were missed this month; download the marks. |
| Teacher | Mark today's attendance; record a mark 4 for a student; export the class marks. |
| Admin | Create a class and a student; add a lesson to the timetable; publish a news item. |
**Metrics:** task success (target >= 90 %), time per task, number of errors, SUS questionnaire (target >= 75).
**SUS questions** (1 strongly disagree - 5 strongly agree): I would like to use the system often; it is unnecessarily complex; it is easy to use; I would need technical help; functions are well integrated; too much inconsistency; most people would learn it quickly; it is cumbersome; I felt confident; I needed to learn a lot first. SUS = ((sum of odd items - 5) + (25 - sum of even items)) x 2.5.
| Participant | Role | Task 1 | Task 2 | Task 3 | Errors | SUS |
|---|---|---|---|---|---|---|
| | | | | | | |

## 5. Security checks
Covered by automated tests: CSRF (missing or foreign Origin -> 403), cookie flags, XSS in news (escaped), CSV formula injection, rate limit, role boundaries. Manual extras before a release: run `govulncheck ./...` and review dependencies.

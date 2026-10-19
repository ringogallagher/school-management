# Testing

## 1. Test levels
| Level | Where | Needs | Run |
|---|---|---|---|
| Unit | `internal/**/_test.go` (grades, export, news, access control, validators) | nothing | `go test ./...` |
| Web/handler | `internal/web/*_test.go` (CSRF, sessions, rendering, escaping) | nothing | `go test ./...` |
| **Integration over HTTP, in memory** | `internal/app/api_memory_test.go` | nothing | `go test ./internal/app/` |
| **Integration over HTTP, PostgreSQL** | `internal/app/api_pg_test.go` | migrated database | `TEST_DATABASE_URL=postgres://... go test -tags integration ./internal/app/` |
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

Reference run of the tool itself against the in-memory build on a development VM (it measures the HTTP layer only, no database): 12-21 thousand req/s per endpoint, p95 below 3 ms, 0 % errors. Real numbers with PostgreSQL must be measured on the target machine.

| Endpoint | Workers | req/s | p50 | p95 | p99 | Errors |
|---|---|---|---|---|---|---|
| | | | | | | |

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

# Project Structure Reference

## Current layout (after Phase 5)

```
cmd/
  api/            HTTP server (API under /api, HTML frontend at /)
  seed/           demo data loader
internal/
  db/             connection pool
  models/         entities and DTOs
  repository/     PostgreSQL queries (one file per entity)
  services/       business logic, access control, jobs (+ *_test.go with in-memory mocks)
  handlers/       JSON API handlers
  web/            HTMX frontend: Go handlers, templates/, static/
  middleware/     JWT auth, RequireRole, rate limiting, security headers
  mailer/         e-mail delivery
migrations/       000001 schema, 000002 constraints & indexes
.github/workflows CI pipeline
```

### Files added in Phases 3–5 worth knowing

| File | Role |
|---|---|
| `cmd/api/main.go` | server: configuration, wiring, API + web routes, background jobs |
| `cmd/seed/main.go` | loads a demo school into an empty database |
| `internal/web/web.go` | frontend core: templates, cookie sessions, CSRF origin check, routes |
| `internal/web/pages_common.go` | login, dashboards, notifications, account, timetable |
| `internal/web/pages_teaching.go` | attendance sheet & stats, gradebook, student report pages |
| `internal/web/pages_admin.go` | admin pages: users, classes, subjects, teachers, students, parents, timetable, announcements |
| `internal/web/templates/` | html/template pages (layout.html = shell, navigation, week grid) |
| `internal/web/static/` | app.css and vendored htmx.min.js (embedded into the binary) |
| `internal/services/access.go` | AccessControl: who may see/change which class or student |
| `internal/services/user_service.go` | admin user management, bootstrap admin |
| `internal/services/attendance_service.go` | attendance marking, reports, low-attendance warnings |
| `internal/services/grade_service.go` | grades, averages, GPA |
| `internal/services/timetable_service.go` | timetable with clash detection and teacher qualifications |
| `internal/services/notification_service.go` | in-app notifications + optional e-mail copies |
| `internal/services/reminder_service.go` | daily "attendance not marked" reminder job |
| `internal/services/subject_service.go` | subjects and teacher subject assignments |
| `internal/services/parent_service.go` | parent accounts and parent-student links |
| `internal/middleware/security.go` | security headers, login rate limiter |
| `internal/mailer/mailer.go` | SMTP and log-only mailers |
| `migrations/000002_academic_constraints.up.sql` | timetable time CHECK, report indexes |
| `.github/workflows/ci.yml` | CI: lint, tests, PostgreSQL integration, Docker build |

---

## Original Phase 1 reference

The rest of this document describes the files as they were created in Phase 1.


## Directory Tree

```
school-management/
│
├── cmd/                                    # Command-line applications
│   └── api/
│       └── main.go                        # API server entry point
│
├── internal/                              # Private application code
│   ├── db/
│   │   └── db.go                         # Database connection & initialization
│   │
│   ├── handlers/                         # HTTP request handlers
│   │   └── auth_handler.go               # Authentication endpoints
│   │
│   ├── middleware/                       # HTTP middleware
│   │   └── auth.go                       # JWT authentication & RBAC middleware
│   │
│   ├── models/                           # Data structures
│   │   └── models.go                     # User, Student, Teacher, etc types
│   │
│   ├── repository/                       # Data access layer
│   │   └── user_repository.go            # User database operations (CRUD)
│   │
│   └── services/                         # Business logic layer
│       ├── auth_service.go               # Authentication logic (register, login, JWT)
│       └── auth_service_test.go          # Unit tests for auth service
│
├── migrations/                            # Database migrations
│   ├── 000001_init_schema.up.sql         # Schema creation (all tables)
│   └── 000001_init_schema.down.sql       # Schema cleanup (rollback)
│
├── tests/                                 # Integration tests
│   └── (tests will be added here)
│
├── .env.example                          # Environment variables template
├── .gitignore                            # Git ignore rules
├── API_TESTING.md                        # API examples with curl
├── docker-compose.yml                    # Docker container setup
├── Dockerfile                            # Application Docker image
├── go.mod                                # Go module definition
├── Makefile                              # Development commands
├── PROJECT_STRUCTURE.md                  # This file
├── README.md                             # Project overview
└── SETUP_GUIDE.md                        # Detailed setup instructions
```

## File Descriptions

### Core Application Files

#### `cmd/api/main.go`
- Application entry point
- Initializes logger, database, services, handlers
- Sets up HTTP router (Chi)
- Configures middleware
- Starts server on configured port

#### `internal/db/db.go`
- PostgreSQL connection management
- Connection pooling configuration
- Health check implementation
- Transaction support

#### `internal/models/models.go`
- Data structures for all entities:
  - User (base for all roles)
  - Student, Teacher, Parent
  - Subject, Class
  - Attendance, Grade, Timetable
  - Notification, Session
- Request/Response DTOs
- Enums (UserRole, AttendanceStatus)

### Handler Layer

#### `internal/handlers/auth_handler.go`
- Register endpoint
- Login endpoint
- Get profile endpoint
- Change password endpoint
- Helper functions for JSON response/error handling

**Endpoints:**
```
POST   /api/auth/register
POST   /api/auth/login
GET    /api/auth/profile
POST   /api/auth/change-password
```

### Service Layer

#### `internal/services/auth_service.go`
- User registration logic
- Login & password verification
- JWT token generation & validation
- Password hashing (bcrypt)
- Password change logic

**Methods:**
```go
Register()           // Create new user account
Login()             // Authenticate user, return JWT
GenerateToken()     // Create JWT for user
ValidateToken()     // Verify & parse JWT
ChangePassword()    // Update user password
```

#### `internal/services/auth_service_test.go`
- Unit tests for auth service
- Mock repository implementation
- Test cases for all auth functions
- Error scenario testing

### Repository Layer

#### `internal/repository/user_repository.go`
- Database CRUD operations for users
- Query builders
- Transaction handling

**Methods:**
```go
Create()    // Insert new user
GetByID()   // Fetch user by ID
GetByEmail() // Fetch user by email
Update()    // Update user record
Delete()    // Soft delete user
GetByRole() // Fetch users by role
```

### Middleware

#### `internal/middleware/auth.go`
- JWT verification middleware
- Role-based access control (RBAC)
- Context value management
- Authorization checks

**Functions:**
```go
AuthMiddleware()    // Validate JWT token
RequireRole()       // Check user has required role
GetUserClaims()     // Extract claims from request
```

### Database Migrations

#### `migrations/000001_init_schema.up.sql`
- Creates all database tables
- Defines column types & constraints
- Sets up foreign keys
- Creates indexes for performance
- Defines enums (user_role, attendance_status)

**Tables:**
```
users
teachers
students
parents
subjects
classes
student_parents
teacher_subjects
attendance
grades
timetable
notifications
sessions
```

#### `migrations/000001_init_schema.down.sql`
- Drops all tables (reverse order)
- Cleans up types/enums
- Used for rollback

### Configuration & Build

#### `go.mod`
- Go module definition
- Dependency versions
- Minimum Go version (1.21)

#### `Dockerfile`
- Multi-stage build (builder + runtime)
- Minimal Alpine Linux image
- Build optimization with CGO_ENABLED=0

#### `docker-compose.yml`
- PostgreSQL service
- API service
- Database migrations service
- Volume definitions
- Network configuration
- Health checks

#### `Makefile`
- Build commands
- Run commands
- Database migration commands
- Docker commands
- Testing & formatting commands

#### `.env.example`
- Template for environment variables
- Documentation of required settings

#### `.gitignore`
- Excludes build artifacts
- Excludes IDE files
- Excludes .env files
- Excludes dependencies

### Documentation

#### `README.md`
- Project overview
- Features list
- Tech stack
- Quick start (Docker & local)
- API documentation
- Database schema overview
- Development guidelines

#### `SETUP_GUIDE.md`
- Step-by-step setup instructions
- Option 1: Docker (recommended)
- Option 2: Local development
- Troubleshooting section
- Production considerations

#### `API_TESTING.md`
- curl examples for all endpoints
- Postman setup guide
- Testing workflow
- Response codes reference
- Troubleshooting guide

#### `PROJECT_STRUCTURE.md`
- This file
- Complete file descriptions
- Code organization explanation

## Code Organization Principles

### Layered Architecture

```
┌─────────────────────────────┐
│     HTTP Layer              │  ← Handlers (request/response)
│  (handlers/)                │
├─────────────────────────────┤
│    Business Logic           │  ← Services (auth, validation)
│     (services/)             │
├─────────────────────────────┤
│     Data Access             │  ← Repository (queries)
│    (repository/)            │
├─────────────────────────────┤
│     Database                │  ← PostgreSQL
│     (db/)                   │
└─────────────────────────────┘
```

### Separation of Concerns

- **Handlers**: Only HTTP logic (parsing, response formatting)
- **Services**: Only business logic (auth, validation, calculation)
- **Repository**: Only database operations (queries, CRUD)
- **Models**: Only data structures (no logic)
- **Middleware**: Only cross-cutting concerns (auth, logging)

## Key Design Patterns

### 1. Repository Pattern
- Abstracts database operations
- Easy to mock for testing
- Single responsibility

### 2. Service Layer
- Encapsulates business logic
- Reusable across handlers
- Testable in isolation

### 3. Middleware Chain
- Request preprocessing
- Authentication enforcement
- Authorization checks

### 4. Error Handling
- Explicit error checking
- Wrapped errors with context
- Graceful error responses

### 5. Context Usage
- Pass values through request
- Cancellation support
- Timeout handling

## Extending the Project

### To Add a New Feature (e.g., Students):

1. **Create Repository**: `internal/repository/student_repository.go`
   - Implement CRUD operations
   - Query builders

2. **Create Service**: `internal/services/student_service.go`
   - Business logic
   - Validation
   - Error handling

3. **Create Handler**: `internal/handlers/student_handler.go`
   - HTTP endpoints
   - Request/response formatting

4. **Add Tests**: `internal/services/student_service_test.go`
   - Unit tests for service

5. **Create Migration**: `migrations/000002_add_student_features.up.sql`
   - Any new tables/columns

6. **Update Main**: `cmd/api/main.go`
   - Register routes
   - Inject dependencies

## Database Design Principles

- **Normalization**: Tables split appropriately (3NF)
- **Foreign Keys**: Maintain referential integrity
- **Indexes**: On frequently queried columns
- **Soft Deletes**: deleted_at column for data preservation
- **Timestamps**: created_at, updated_at on all entities
- **Constraints**: Check constraints for valid data

## Testing Strategy

### Unit Tests
- Test individual services
- Mock repositories
- Table-driven tests

### Integration Tests
- Test handlers with mock DB
- Test middleware
- End-to-end workflows

### Load Testing
- Performance under concurrent users
- Database connection pooling

## Next Files to Create

As you extend the project, create:

1. `internal/repository/student_repository.go` - Student CRUD
2. `internal/repository/class_repository.go` - Class CRUD
3. `internal/repository/attendance_repository.go` - Attendance CRUD
4. `internal/repository/grade_repository.go` - Grade CRUD
5. `internal/services/student_service.go` - Student logic
6. `internal/handlers/student_handler.go` - Student endpoints
7. ... (similar for other features)
8. `cmd/api/templates/` - HTML templates for HTMX frontend
9. `.github/workflows/ci.yml` - CI/CD pipeline

## Summary Stats

- **Total Files**: 20+
- **Lines of Code**: ~2000+ (without comments)
- **Go Packages**: 6 (db, handlers, middleware, models, repository, services)
- **Database Tables**: 13
- **API Endpoints**: 4 (auth endpoints, with 7 more placeholder routes)
- **Test Cases**: 10+ unit tests

This structure provides a solid foundation for a production-grade web application!

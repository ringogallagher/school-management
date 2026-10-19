# Complete Setup Guide

This guide will walk you through setting up the School Management System from scratch.

## Prerequisites

- Git
- Docker & Docker Compose (OR PostgreSQL + Go 1.21+)
- curl or Postman (for testing API)
- Code editor (VS Code recommended)

## Option 1: Docker Setup (Recommended for Beginners)

### Step 1: Clone the Repository

```bash
cd ~/projects  # or your preferred directory
git init school-management
cd school-management
```

Copy all the files from the project structure into this directory.

### Step 2: Start Docker Containers

```bash
docker-compose up -d
```

This will:
- Start PostgreSQL database
- Run database migrations automatically
- Start the API server

### Step 3: Verify Everything is Running

```bash
# Check containers are running
docker-compose ps

# Test the API
curl http://localhost:8080/health
# Expected response: {"status":"ok"}
```

### Step 4: Test the API

Follow the API testing examples in `API_TESTING.md`

### Step 5: View Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f api
docker-compose logs -f postgres
```

## Option 2: Local Development Setup

### Step 1: Install Dependencies

**macOS:**
```bash
# Install Go (if not already installed)
brew install go

# Install PostgreSQL
brew install postgresql

# Install golang-migrate
brew install golang-migrate
```

**Ubuntu/Debian:**
```bash
# Install Go
sudo apt-get install golang-go

# Install PostgreSQL
sudo apt-get install postgresql postgresql-contrib

# Install golang-migrate
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.16.2/migrate.linux-amd64.tar.gz | tar xvz
sudo mv migrate /usr/local/bin/
```

**Windows:**
- Download Go from https://golang.org/dl/
- Download PostgreSQL from https://www.postgresql.org/download/windows/
- Download golang-migrate from https://github.com/golang-migrate/migrate/releases

### Step 2: Setup PostgreSQL

```bash
# Start PostgreSQL service (it should auto-start on macOS with Homebrew)
# If not running, start it manually:
# macOS: brew services start postgresql
# Ubuntu: sudo service postgresql start

# Create database
createdb school_management

# Verify connection
psql school_management -c "SELECT 1;"
```

### Step 3: Setup Go Project

```bash
# Create project directory
mkdir school-management
cd school-management

# Initialize Git
git init

# Copy all project files into this directory
# (from the provided structure)

# Install Go dependencies
go mod download
```

### Step 4: Run Migrations

```bash
export DATABASE_URL="postgres://$(whoami)@localhost/school_management?sslmode=disable"

# Run migrations up
golang-migrate -path ./migrations -database "$DATABASE_URL" up

# Verify migrations ran successfully
psql school_management -c "\dt"
```

### Step 5: Set Environment Variables

Create a `.env` file in the project root:

```bash
cat > .env << 'EOF'
DATABASE_URL=postgres://$(whoami)@localhost/school_management?sslmode=disable
JWT_SECRET=your-secret-key-change-in-production
PORT=8080
ENVIRONMENT=development
EOF
```

Then export them:
```bash
export $(cat .env | xargs)
```

### Step 6: Build and Run

```bash
# Build
make build

# Run
make run

# Or run directly
go run cmd/api/main.go
```

### Step 7: Test the API

```bash
# In another terminal
curl http://localhost:8080/health
```

## Project Structure Overview

```
school-management/
├── cmd/api/
│   └── main.go                 # Application entry point
├── internal/
│   ├── db/
│   │   └── db.go               # Database connection
│   ├── handlers/
│   │   └── auth_handler.go     # HTTP request handlers
│   ├── middleware/
│   │   └── auth.go             # JWT authentication middleware
│   ├── models/
│   │   └── models.go           # Data structures
│   ├── repository/
│   │   └── user_repository.go  # Database queries
│   └── services/
│       ├── auth_service.go     # Business logic
│       └── auth_service_test.go # Unit tests
├── migrations/
│   ├── 000001_init_schema.up.sql    # Schema creation
│   └── 000001_init_schema.down.sql  # Schema rollback
├── tests/                      # Integration tests
├── docker-compose.yml
├── Dockerfile
├── Makefile
├── go.mod                      # Go dependencies
├── README.md
├── SETUP_GUIDE.md              # This file
└── API_TESTING.md              # API examples
```

## Understanding the Architecture

### Layers

1. **Handlers** - HTTP request/response handling
2. **Services** - Business logic (auth, validation, etc)
3. **Repository** - Database operations
4. **Models** - Data structures

### Data Flow

```
HTTP Request
    ↓
Middleware (auth check, logging)
    ↓
Handler (parse request)
    ↓
Service (business logic)
    ↓
Repository (query database)
    ↓
Response (JSON)
```

## Next Steps

### 1. Test Authentication

```bash
# The first admin is created on startup from ADMIN_EMAIL / ADMIN_PASSWORD.
# Load a demo school with teachers, students, parents, timetable, grades and attendance:
make seed            # local;  in Docker: make docker-seed
# Then open http://localhost:8080 in the browser, or use the API:

# Login
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@school.test",
    "password": "admin12345"
  }'

# Get profile (use token from login response)
curl -X GET http://localhost:8080/api/auth/profile \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

### 2. Implement Student Management

Create these files:
- `internal/repository/student_repository.go` - Database operations
- `internal/services/student_service.go` - Business logic
- `internal/handlers/student_handler.go` - HTTP endpoints

### 3. Implement Attendance Management

Similar structure:
- `internal/repository/attendance_repository.go`
- `internal/services/attendance_service.go`
- `internal/handlers/attendance_handler.go`

### 4. Create Frontend with HTMX

Only after API is ready:
- Create `cmd/api/templates/` directory
- Add HTML templates for each view
- Update handlers to serve templates
- Add HTMX for dynamic interactions

## Troubleshooting

### Port Already in Use

```bash
# Check what's using port 8080
lsof -i :8080

# Kill the process (macOS/Linux)
kill -9 <PID>

# Or use different port
PORT=8081 go run cmd/api/main.go
```

### Database Connection Error

```bash
# Check PostgreSQL is running
psql postgres -c "SELECT 1;"

# Create database if it doesn't exist
createdb school_management

# Check connection string
echo $DATABASE_URL
```

### Migration Error

```bash
# Check migration status
migrate -path ./migrations -database "$DATABASE_URL" version

# Rollback all
migrate -path ./migrations -database "$DATABASE_URL" down -all

# Re-run migrations
migrate -path ./migrations -database "$DATABASE_URL" up
```

### Import Not Found

```bash
# Update go.mod
go mod tidy

# Download dependencies
go mod download
```

## Useful Commands

```bash
# Format code
make fmt
# or
go fmt ./...

# Run tests
make test
# or
go test -v ./...

# Check for issues
go vet ./...

# Build for production
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/api cmd/api/main.go
```

## Docker Useful Commands

```bash
# See container logs
docker-compose logs -f

# Connect to database
docker-compose exec postgres psql -U postgres -d school_management

# Rebuild containers
docker-compose up --build

# Stop containers
docker-compose down

# Remove everything including volumes
docker-compose down -v
```

## Production Considerations

Before deploying to production:

1. **Change JWT_SECRET**: Use a strong, random secret
2. **Setup SSL/TLS**: Use proper certificates
3. **Database**: Use managed PostgreSQL (RDS, Cloud SQL, etc)
4. **Environment**: Use proper environment variable management
5. **Logging**: Setup centralized logging
6. **Monitoring**: Add application monitoring
7. **Backups**: Setup regular database backups
8. **Security**: Add rate limiting, CORS policies, etc

## Getting Help

If you encounter issues:

1. Check the error message carefully
2. Check `docker-compose logs` or application logs
3. Verify all prerequisites are installed
4. Check database connection
5. Review API_TESTING.md for expected responses
6. Open an issue in the repository

## Next Learning Goals

1. ✅ Understand project structure
2. ✅ Setup local development environment
3. ⏭️ Learn Go basics (functions, interfaces, error handling)
4. ⏭️ Understand database design
5. ⏭️ Learn HTTP REST API design
6. ⏭️ Implement student management features
7. ⏭️ Write tests
8. ⏭️ Create HTMX frontend

Good luck with your thesis project! 🚀

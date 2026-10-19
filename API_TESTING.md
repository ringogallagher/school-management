# API Testing Guide

This guide provides curl examples for testing the School Management API.

## Base URL

```
http://localhost:8080/api
```

## Health Check

```bash
curl -X GET http://localhost:8080/health
```

## Authentication Endpoints

### 1. Your first login (and creating accounts)

Public self-registration is **disabled** — previously anyone could register themselves as an admin.
The first administrator is created on startup from `ADMIN_EMAIL` / `ADMIN_PASSWORD`
(docker-compose sets `admin@school.test` / `admin12345`). For a full demo school run `make seed`
(or `make docker-seed`), which prints the demo logins.

Teachers, students and parents are created with their profile via `/api/teachers`, `/api/students`
and `/api/parents`. Other administrators can be created by an admin:

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Authorization: Bearer $ADMIN" -H "Content-Type: application/json" \
  -d '{"email": "office@school.test", "password": "password123", "first_name": "Office", "role": "admin"}'
# Without an admin token this returns 401. Passwords must be at least 8 characters.
```

### 2. Login

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "student@school.com",
    "password": "student123"
  }'
```

**Response:**
```json
{
  "code": 200,
  "message": "login successful",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": 1,
      "email": "student@school.com",
      "first_name": "Jane",
      "last_name": "Student",
      "role": "student",
      "is_active": true
    },
    "expires_at": "2024-01-16T10:30:00Z"
  }
}
```

Save the token for use in other requests:
```bash
export TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### 3. Get User Profile

```bash
curl -X GET http://localhost:8080/api/auth/profile \
  -H "Authorization: Bearer $TOKEN"
```

### 4. Change Password

```bash
curl -X POST http://localhost:8080/api/auth/change-password \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "old_password": "student123",
    "new_password": "newpassword456"
  }'
```

## Class, Teacher & Student Endpoints (Phase 2)

All of these require `Authorization: Bearer $TOKEN`. Reads (`GET`) work for
`admin` and `teacher` tokens; writes (`POST`/`PUT`/`DELETE`) require an
`admin` token. Classes are readable by any authenticated user.

### Create a teacher (creates both the login and the profile)

```bash
curl -X POST http://localhost:8080/api/teachers \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "teacher@school.com",
    "password": "teacher123",
    "first_name": "John",
    "last_name": "Teacher",
    "department": "Mathematics",
    "specialization": "Algebra",
    "hire_date": "2023-09-01"
  }'
```

Note the returned `user_id` — that's the id you pass as `teacher_id` when
creating a class.

### Create a class (needs an existing teacher's user id)

```bash
curl -X POST http://localhost:8080/api/classes \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Homeroom 9A",
    "level": "9A",
    "section": "A",
    "teacher_id": 2,
    "max_students": 30
  }'
```

### Create a student (creates both the login and the profile)

```bash
curl -X POST http://localhost:8080/api/students \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "student@school.com",
    "password": "student123",
    "first_name": "Jane",
    "last_name": "Student",
    "class_id": 1,
    "student_number": "S-2026-001",
    "enrollment_date": "2026-09-01"
  }'
```

### List / get / update / delete

```bash
# List (pagination via ?limit=&offset=)
curl -H "Authorization: Bearer $ADMIN_TOKEN" "http://localhost:8080/api/classes?limit=20&offset=0"
curl -H "Authorization: Bearer $ADMIN_TOKEN" "http://localhost:8080/api/teachers"
curl -H "Authorization: Bearer $ADMIN_TOKEN" "http://localhost:8080/api/students"
curl -H "Authorization: Bearer $ADMIN_TOKEN" "http://localhost:8080/api/students/class/1"

# Get by id
curl -H "Authorization: Bearer $ADMIN_TOKEN" "http://localhost:8080/api/classes/1"

# Update
curl -X PUT http://localhost:8080/api/classes/1 \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" \
  -d '{"max_students": 25}'

curl -X PUT http://localhost:8080/api/students/1 \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" \
  -d '{"class_id": 2}'

# Delete
curl -X DELETE http://localhost:8080/api/classes/1 -H "Authorization: Bearer $ADMIN_TOKEN"
curl -X DELETE http://localhost:8080/api/teachers/1 -H "Authorization: Bearer $ADMIN_TOKEN"
curl -X DELETE http://localhost:8080/api/students/1 -H "Authorization: Bearer $ADMIN_TOKEN"
```

### Verify RBAC

```bash
# Should fail with 403 - a student token cannot create a class
curl -X POST http://localhost:8080/api/classes \
  -H "Authorization: Bearer $STUDENT_TOKEN" -H "Content-Type: application/json" \
  -d '{"name": "X", "level": "9A", "teacher_id": 2}'
```

## Academic Endpoints (Phase 3)

The examples assume `$ADMIN`, `$TEACHER`, `$STUDENT`, `$PARENT` hold tokens from `/api/auth/login`.

### Subjects (admin manages, everyone can read)
```bash
curl -X POST http://localhost:8080/api/subjects \
  -H "Authorization: Bearer $ADMIN" -H "Content-Type: application/json" \
  -d '{"name": "Mathematics", "code": "math", "credits": 4}'   # code stored as MATH
curl http://localhost:8080/api/subjects -H "Authorization: Bearer $TEACHER"
```

### Parents
```bash
# Create parent (login + profile), then link to student 1
curl -X POST http://localhost:8080/api/parents \
  -H "Authorization: Bearer $ADMIN" -H "Content-Type: application/json" \
  -d '{"email": "mama@school.edu", "password": "password123", "first_name": "Maria", "relationship": "mother"}'
curl -X POST http://localhost:8080/api/parents/1/students/1 -H "Authorization: Bearer $ADMIN"

# As the parent
curl http://localhost:8080/api/parents/me/children -H "Authorization: Bearer $PARENT"
```

### Timetable (admin manages)
`teacher_id` is the **teacher profile id** (from `/api/teachers`), `day_of_week` 0 = Sunday ... 6 = Saturday.
```bash
curl -X POST http://localhost:8080/api/timetable \
  -H "Authorization: Bearer $ADMIN" -H "Content-Type: application/json" \
  -d '{"class_id": 1, "subject_id": 1, "teacher_id": 1, "day_of_week": 1,
       "start_time": "08:00", "end_time": "08:45", "room_number": "101"}'
# Re-sending an overlapping lesson for the same class/teacher/room returns 400 "scheduling conflict: ..."

curl http://localhost:8080/api/timetable/class/1   -H "Authorization: Bearer $STUDENT"
curl http://localhost:8080/api/timetable/teacher/1 -H "Authorization: Bearer $ADMIN"
curl http://localhost:8080/api/timetable/me        -H "Authorization: Bearer $TEACHER"
```

### Attendance (admin / teacher of that class)
```bash
# Mark a whole class for a day (date optional, defaults to today; status defaults to present)
curl -X POST http://localhost:8080/api/attendance \
  -H "Authorization: Bearer $TEACHER" -H "Content-Type: application/json" \
  -d '{"class_id": 1, "date": "2026-10-06",
       "records": [{"student_id": 1, "status": "present"},
                   {"student_id": 2, "status": "absent", "remarks": "ill"}]}'

curl "http://localhost:8080/api/attendance/class/1?date=2026-10-06"                  -H "Authorization: Bearer $TEACHER"
curl "http://localhost:8080/api/attendance/class/1/stats?from=2026-09-01&to=2026-10-06" -H "Authorization: Bearer $TEACHER"
curl "http://localhost:8080/api/attendance/student/1"   -H "Authorization: Bearer $PARENT"   # own child only
curl  http://localhost:8080/api/attendance/me           -H "Authorization: Bearer $STUDENT"
```

### Grades
```bash
curl -X POST http://localhost:8080/api/grades \
  -H "Authorization: Bearer $TEACHER" -H "Content-Type: application/json" \
  -d '{"student_id": 1, "subject_id": 1, "marks": 5, "term": "Q1"}'

curl -X PUT http://localhost:8080/api/grades/1 \
  -H "Authorization: Bearer $TEACHER" -H "Content-Type: application/json" -d '{"marks": 4}'

curl "http://localhost:8080/api/grades/student/1?term=Q1"          -H "Authorization: Bearer $PARENT"   # grades + subject averages + final marks + overall average
curl  http://localhost:8080/api/grades/me                          -H "Authorization: Bearer $STUDENT"
curl "http://localhost:8080/api/grades/class/1/subject/1?term=Q1"  -H "Authorization: Bearer $TEACHER"
```

### Notifications (every user, own inbox)
```bash
curl "http://localhost:8080/api/notifications?unread=true" -H "Authorization: Bearer $PARENT"
curl  http://localhost:8080/api/notifications/unread-count -H "Authorization: Bearer $PARENT"
curl -X POST http://localhost:8080/api/notifications/5/read -H "Authorization: Bearer $PARENT"
curl -X POST http://localhost:8080/api/notifications/read-all -H "Authorization: Bearer $PARENT"

# Admin announcement (role optional: admin | teacher | student | parent)
curl -X POST http://localhost:8080/api/notifications/announce \
  -H "Authorization: Bearer $ADMIN" -H "Content-Type: application/json" \
  -d '{"role": "teacher", "title": "Staff meeting", "message": "Friday 15:00, room 12"}'
```

### Who can do what
| Endpoint | admin | teacher | student | parent |
|---|---|---|---|---|
| Manage subjects / timetable / parents | ✅ | ❌ | ❌ | ❌ |
| Mark attendance, record grades | ✅ | own classes* | ❌ | ❌ |
| Edit / delete a grade | ✅ | only grades they recorded | ❌ | ❌ |
| Student attendance & grade reports | ✅ | own classes* | self | linked children |
| `/me` timetable | – | own lessons | class | – |

\* "own classes" = homeroom class **or** any class they have a timetable lesson with.

## User management, teacher subjects & security (Phase 4/5)

```bash
# Search users (role and free-text filter on name/email)
curl "http://localhost:8080/api/users?role=teacher&q=kovacs" -H "Authorization: Bearer $ADMIN"

# Edit / deactivate (admins can't deactivate or delete themselves)
curl -X PUT http://localhost:8080/api/users/5 -H "Authorization: Bearer $ADMIN" \
  -H "Content-Type: application/json" -d '{"is_active": false}'
curl -X POST http://localhost:8080/api/users/5/reset-password -H "Authorization: Bearer $ADMIN" \
  -H "Content-Type: application/json" -d '{"new_password": "temporary123"}'
curl -X DELETE http://localhost:8080/api/users/5 -H "Authorization: Bearer $ADMIN"   # soft delete

# Subjects a teacher is qualified for (teacher profile id). Once set, the timetable
# refuses to schedule that teacher for other subjects. An empty list removes the restriction.
curl -X PUT http://localhost:8080/api/teachers/1/subjects -H "Authorization: Bearer $ADMIN" \
  -H "Content-Type: application/json" -d '{"subject_ids": [1, 2]}'
curl http://localhost:8080/api/teachers/1/subjects -H "Authorization: Bearer $TEACHER"
```

- **Login rate limit:** more than `LOGIN_RATE_LIMIT` (default 10) login attempts per minute from one IP
  return `429 Too Many Requests` with a `Retry-After` header.
- **Attendance reminders:** every day at `REMINDER_HOUR` (default 15:00) homeroom teachers of classes
  that had lessons but no attendance get a notification (and e-mail, if configured).
- **E-mail:** with `SMTP_HOST` set, grade/absence/low-attendance notifications and reminders are also
  e-mailed; `EMAIL_LOG=true` writes them to the server log instead.

## Testing Workflow

1. **Start the application:**
   ```bash
   make docker-up
   ```

2. **Load the demo school:** `make docker-seed` (prints admin / teacher / student / parent logins)

3. **Test login for each user:**
   - Login with each user and save their tokens

4. **Test protected endpoints:**
   - Use the tokens to access protected endpoints
   - Verify role-based access control works

5. **Test role restrictions:**
   - Try to access admin endpoints with student token (should fail)
   - Try to access teacher endpoints with student token (should fail)

## Using Postman

1. Create a new Postman collection
2. Add the endpoints above
3. Set up environment variables:
   - `base_url`: http://localhost:8080
   - `token`: (will be updated after login)

4. In the Login request, add a post-request script:
   ```javascript
   if (pm.response.code === 200) {
     var jsonData = pm.response.json();
     pm.environment.set("token", jsonData.data.token);
   }
   ```

5. Use `{{token}}` in the Authorization header of protected requests

## Common Response Codes

- **200**: Success
- **201**: Created
- **400**: Bad Request (invalid input)
- **401**: Unauthorized (missing/invalid token)
- **403**: Forbidden (insufficient permissions)
- **404**: Not Found
- **500**: Internal Server Error

## Troubleshooting

### Database connection error
- Ensure PostgreSQL is running
- Check DATABASE_URL environment variable
- Verify database exists: `psql -c "CREATE DATABASE school_management;"`

### Migration error
- Run: `make migrate-up`
- Check migration files in `migrations/` directory

### JWT token invalid
- Ensure JWT_SECRET is set and consistent
- Check token hasn't expired (24 hours)
- Check Authorization header format: `Authorization: Bearer <token>`

### Role-based access denied
- Verify user role matches endpoint requirements
- Check user was created with correct role
- Ensure token is from correct user account

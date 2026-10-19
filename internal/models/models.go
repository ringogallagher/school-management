package models

import (
	//"database/sql"
	"time"
)

// UserRole defines the role of a user in the system
type UserRole string

const (
	RoleAdmin   UserRole = "admin"
	RoleTeacher UserRole = "teacher"
	RoleStudent UserRole = "student"
	RoleParent  UserRole = "parent"
)

// AttendanceStatus defines attendance status
type AttendanceStatus string

const (
	AttendancePresent AttendanceStatus = "present"
	AttendanceAbsent  AttendanceStatus = "absent"
	AttendanceLate    AttendanceStatus = "late"
	AttendanceExcused AttendanceStatus = "excused"
)

// User represents a user in the system
type User struct {
	ID           int64      `json:"id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"` // Never expose password hash
	FirstName    string     `json:"first_name"`
	LastName     string     `json:"last_name"`
	Role         UserRole   `json:"role"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

// Subject represents a school subject
type Subject struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	Credits     int       `json:"credits"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Class represents a school class
type Class struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Level       string    `json:"level"` // e.g., 9A, 10B
	Section     string    `json:"section"`
	TeacherID   int64     `json:"teacher_id"`
	MaxStudents int       `json:"max_students"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Teacher represents a teacher
type Teacher struct {
	ID             int64      `json:"id"`
	UserID         int64      `json:"user_id"`
	Department     string     `json:"department"`
	Specialization string     `json:"specialization"`
	HireDate       *time.Time `json:"hire_date"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Student represents a student
type Student struct {
	ID             int64     `json:"id"`
	UserID         int64     `json:"user_id"`
	ClassID        int64     `json:"class_id"`
	EnrollmentDate time.Time `json:"enrollment_date"`
	StudentNumber  string    `json:"student_number"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Parent represents a parent/guardian
type Parent struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	Relationship string    `json:"relationship"` // mother, father, guardian, etc
	PhoneNumber  string    `json:"phone_number"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Attendance represents attendance record
type Attendance struct {
	ID         int64            `json:"id"`
	StudentID  int64            `json:"student_id"`
	ClassID    int64            `json:"class_id"`
	Date       time.Time        `json:"date"`
	Status     AttendanceStatus `json:"status"`
	Remarks    string           `json:"remarks"`
	RecordedBy *int64           `json:"recorded_by"` // teacher ID
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
}

// Grade represents a grade/mark
type Grade struct {
	ID         int64     `json:"id"`
	StudentID  int64     `json:"student_id"`
	SubjectID  int64     `json:"subject_id"`
	ClassID    int64     `json:"class_id"`
	Marks      float64   `json:"marks"`
	Term       string    `json:"term"`        // Q1, Q2, Final, etc
	RecordedBy int64     `json:"recorded_by"` // teacher ID
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Timetable represents a timetable entry
type Timetable struct {
	ID         int64     `json:"id"`
	ClassID    int64     `json:"class_id"`
	SubjectID  int64     `json:"subject_id"`
	TeacherID  int64     `json:"teacher_id"`
	DayOfWeek  int       `json:"day_of_week"` // 0=Sunday, 6=Saturday
	StartTime  string    `json:"start_time"`  // HH:MM format
	EndTime    string    `json:"end_time"`    // HH:MM format
	RoomNumber string    `json:"room_number"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Notification represents a notification
type Notification struct {
	ID        int64      `json:"id"`
	UserID    int64      `json:"user_id"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Message   string     `json:"message"`
	IsRead    bool       `json:"is_read"`
	ReadAt    *time.Time `json:"read_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// Session represents a user session
type Session struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	TokenHash string    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// ============ DTO / Request/Response Models ============

// RegisterRequest for user registration
type RegisterRequest struct {
	Email     string   `json:"email"`
	Password  string   `json:"password"`
	FirstName string   `json:"first_name"`
	LastName  string   `json:"last_name"`
	Role      UserRole `json:"role"`
}

// LoginRequest for user login
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse contains JWT token
type AuthResponse struct {
	Token     string    `json:"token"`
	User      *User     `json:"user"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ErrorResponse for API errors
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// SuccessResponse for successful operations
type SuccessResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ============ Class DTOs ============

// CreateClassRequest for creating a new class
type CreateClassRequest struct {
	Name        string `json:"name"`
	Level       string `json:"level"`
	Section     string `json:"section"`
	TeacherID   int64  `json:"teacher_id"`
	MaxStudents int    `json:"max_students"`
}

// UpdateClassRequest for updating a class. Zero-value fields are left unchanged.
type UpdateClassRequest struct {
	Name        string `json:"name"`
	Level       string `json:"level"`
	Section     string `json:"section"`
	TeacherID   int64  `json:"teacher_id"`
	MaxStudents int    `json:"max_students"`
}

// ============ Teacher DTOs ============

// CreateTeacherRequest creates both a user account (role=teacher) and a teacher profile
type CreateTeacherRequest struct {
	Email          string `json:"email"`
	Password       string `json:"password"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	Department     string `json:"department"`
	Specialization string `json:"specialization"`
	HireDate       string `json:"hire_date"` // YYYY-MM-DD, optional
}

// UpdateTeacherRequest for updating a teacher profile. Zero-value fields are left unchanged.
type UpdateTeacherRequest struct {
	Department     string `json:"department"`
	Specialization string `json:"specialization"`
}

// TeacherWithUser combines a teacher profile with its linked user account info
type TeacherWithUser struct {
	Teacher
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	IsActive  bool   `json:"is_active"`
}

// ============ Student DTOs ============

// CreateStudentRequest creates both a user account (role=student) and a student profile
type CreateStudentRequest struct {
	Email          string `json:"email"`
	Password       string `json:"password"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	ClassID        int64  `json:"class_id"`
	StudentNumber  string `json:"student_number"`
	EnrollmentDate string `json:"enrollment_date"` // YYYY-MM-DD, optional (defaults to today)
}

// UpdateStudentRequest for updating a student profile (currently: reassigning class)
type UpdateStudentRequest struct {
	ClassID int64 `json:"class_id"`
}

// StudentWithUser combines a student profile with its linked user account info
type StudentWithUser struct {
	Student
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	IsActive  bool   `json:"is_active"`
}

// ============ Subject DTOs ============

// CreateSubjectRequest for creating a subject
type CreateSubjectRequest struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Credits     int    `json:"credits"`
}

// UpdateSubjectRequest for updating a subject. Zero-value fields are left unchanged.
type UpdateSubjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Credits     int    `json:"credits"`
}

// ============ Parent DTOs ============

// CreateParentRequest creates both a user account (role=parent) and a parent profile
type CreateParentRequest struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Relationship string `json:"relationship"`
	PhoneNumber  string `json:"phone_number"`
}

// ParentWithUser combines a parent profile with its linked user account info
type ParentWithUser struct {
	Parent
	Email      string `json:"email"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	ChildCount int    `json:"child_count"`
}

// ============ Attendance DTOs ============

// AttendanceEntry is one student's status inside a bulk attendance request
type AttendanceEntry struct {
	StudentID int64            `json:"student_id"`
	Status    AttendanceStatus `json:"status"`
	Remarks   string           `json:"remarks"`
}

// MarkAttendanceRequest marks attendance for several students of one class on one day.
// Re-submitting for the same class and date overwrites the earlier records.
type MarkAttendanceRequest struct {
	ClassID int64             `json:"class_id"`
	Date    string            `json:"date"` // YYYY-MM-DD, optional (defaults to today)
	Records []AttendanceEntry `json:"records"`
}

// AttendanceWithStudent is an attendance record joined with the student's name
type AttendanceWithStudent struct {
	Attendance
	StudentNumber string `json:"student_number"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
}

// AttendanceSummary aggregates attendance counts for one student over a period.
// Percentage = (present + late) / (total - excused) * 100; excused days do not count against the student.
type AttendanceSummary struct {
	StudentID  int64   `json:"student_id"`
	FirstName  string  `json:"first_name,omitempty"`
	LastName   string  `json:"last_name,omitempty"`
	Total      int     `json:"total"`
	Present    int     `json:"present"`
	Absent     int     `json:"absent"`
	Late       int     `json:"late"`
	Excused    int     `json:"excused"`
	Percentage float64 `json:"percentage"`
}

// StudentAttendanceReport is a student's attendance records plus their summary
type StudentAttendanceReport struct {
	Summary AttendanceSummary `json:"summary"`
	Records []*Attendance     `json:"records"`
}

// ============ Grade DTOs ============

// RecordGradeRequest records a mark for a student in a subject.
// Marks use the five-point scale: whole numbers from 1 (worst) to 5 (best).
type RecordGradeRequest struct {
	StudentID int64   `json:"student_id"`
	SubjectID int64   `json:"subject_id"`
	Marks     float64 `json:"marks"`
	Term      string  `json:"term"`
}

// UpdateGradeRequest updates an existing grade. Nil fields are left unchanged
// (pointers so that a mark of 0 can be set explicitly).
type UpdateGradeRequest struct {
	Marks *float64 `json:"marks"`
	Term  *string  `json:"term"`
}

// GradeWithDetails is a grade joined with subject and student names
type GradeWithDetails struct {
	Grade
	SubjectName string `json:"subject_name"`
	SubjectCode string `json:"subject_code"`
	Credits     int    `json:"credits"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
}

// SubjectAverage is a student's average mark in one subject (five-point scale).
// Final is the average rounded to a whole mark (4.5 and above becomes 5, and so on).
type SubjectAverage struct {
	SubjectID   int64   `json:"subject_id"`
	SubjectName string  `json:"subject_name"`
	Average     float64 `json:"average"`
	Final       int     `json:"final"`
	Count       int     `json:"count"`
}

// StudentGradeReport is a student's grade report: individual marks, per-subject
// averages and the overall average, all on the five-point scale.
type StudentGradeReport struct {
	StudentID      int64               `json:"student_id"`
	Term           string              `json:"term,omitempty"`
	Grades         []*GradeWithDetails `json:"grades"`
	Subjects       []SubjectAverage    `json:"subjects"`
	OverallAverage float64             `json:"overall_average"`
}

// ============ Timetable DTOs ============

// TimetableRequest creates or replaces a timetable entry
type TimetableRequest struct {
	ClassID    int64  `json:"class_id"`
	SubjectID  int64  `json:"subject_id"`
	TeacherID  int64  `json:"teacher_id"` // teacher profile ID (teachers.id)
	DayOfWeek  int    `json:"day_of_week"`
	StartTime  string `json:"start_time"` // HH:MM
	EndTime    string `json:"end_time"`   // HH:MM
	RoomNumber string `json:"room_number"`
}

// TimetableWithDetails is a timetable entry joined with readable names
type TimetableWithDetails struct {
	Timetable
	ClassName   string `json:"class_name"`
	SubjectName string `json:"subject_name"`
	TeacherName string `json:"teacher_name"`
}

// ============ Notification DTOs ============

// AnnouncementRequest broadcasts a system announcement, optionally only to one role
type AnnouncementRequest struct {
	Role    UserRole `json:"role"` // empty = everyone
	Title   string   `json:"title"`
	Message string   `json:"message"`
}

// Notification types
const (
	NotificationGradePosted   = "grade_posted"
	NotificationAttendanceLow = "attendance_low"
	NotificationAbsence       = "absence"
	NotificationAnnouncement  = "announcement"
)

// ============ User management DTOs ============

// UpdateUserRequest is used by admins to edit a user account. Nil fields are left unchanged.
// Role is deliberately not editable: profiles (student/teacher/parent) depend on it.
type UpdateUserRequest struct {
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	Email     *string `json:"email"`
	IsActive  *bool   `json:"is_active"`
}

// ResetPasswordRequest is used by admins to set a new password for a user
type ResetPasswordRequest struct {
	NewPassword string `json:"new_password"`
}

// AssignSubjectsRequest replaces the subjects a teacher is qualified to teach
type AssignSubjectsRequest struct {
	SubjectIDs []int64 `json:"subject_ids"`
}

// ReminderTarget is a class that has lessons today but no attendance recorded yet
type ReminderTarget struct {
	ClassID       int64  `json:"class_id"`
	ClassName     string `json:"class_name"`
	TeacherUserID int64  `json:"teacher_user_id"`
}

// ============ News ============

// News is a school news item shown on the home page
type News struct {
	ID         int64     `json:"id"`
	Title      string    `json:"title"`
	Body       string    `json:"body"`
	Pinned     bool      `json:"pinned"`
	AuthorID   *int64    `json:"author_id"`
	AuthorName string    `json:"author_name"` // filled from the users table when listing / reading
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// NewsRequest creates or replaces a news item
type NewsRequest struct {
	Title  string `json:"title"`
	Body   string `json:"body"`
	Pinned bool   `json:"pinned"`
}

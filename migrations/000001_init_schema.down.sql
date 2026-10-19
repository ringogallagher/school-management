-- Drop all tables in reverse order of creation
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS timetable;
DROP TABLE IF EXISTS grades;
DROP TABLE IF EXISTS attendance;
DROP TABLE IF EXISTS teacher_subjects;
DROP TABLE IF EXISTS student_parents;
DROP TABLE IF EXISTS parents;
DROP TABLE IF EXISTS students;
DROP TABLE IF EXISTS teachers;
DROP TABLE IF EXISTS classes;
DROP TABLE IF EXISTS subjects;
DROP TABLE IF EXISTS users;

-- Drop types
DROP TYPE IF EXISTS attendance_status;
DROP TYPE IF EXISTS user_role;

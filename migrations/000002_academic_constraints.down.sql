DROP INDEX IF EXISTS idx_notifications_user_unread;
DROP INDEX IF EXISTS idx_grades_student_term;
DROP INDEX IF EXISTS idx_attendance_student_date;
DROP INDEX IF EXISTS idx_timetable_room_day;
DROP INDEX IF EXISTS idx_timetable_teacher_day;
ALTER TABLE timetable DROP CONSTRAINT IF EXISTS chk_timetable_time_order;

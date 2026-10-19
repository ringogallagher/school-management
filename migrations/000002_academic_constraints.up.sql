-- Phase 3: constraints and indexes used by attendance, grades and timetable features

-- A lesson must end after it starts
ALTER TABLE timetable
    ADD CONSTRAINT chk_timetable_time_order CHECK (end_time > start_time);

-- Conflict detection looks up all lessons on a given day for a teacher / room
CREATE INDEX idx_timetable_teacher_day ON timetable(teacher_id, day_of_week);
CREATE INDEX idx_timetable_room_day ON timetable(room_number, day_of_week);

-- Attendance reports filter by student and date range
CREATE INDEX idx_attendance_student_date ON attendance(student_id, date);

-- Grade reports filter by student and term
CREATE INDEX idx_grades_student_term ON grades(student_id, term);

-- Listing a user's unread notifications, newest first
CREATE INDEX idx_notifications_user_unread ON notifications(user_id, is_read, created_at DESC);

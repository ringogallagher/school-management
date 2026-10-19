-- Read-only reporting views. They give administrators (and BI tools / psql) ready-made
-- aggregates without re-writing the joins, and keep the percentage rules in one place.

-- Marks per student, subject and term (five-point scale)
CREATE VIEW v_student_grade_summary AS
SELECT g.student_id,
       g.subject_id,
       sub.name                         AS subject_name,
       COALESCE(g.term, '')             AS term,
       COUNT(*)                         AS marks_count,
       ROUND(AVG(g.marks), 2)           AS average_mark
FROM grades g
JOIN subjects sub ON sub.id = g.subject_id
GROUP BY g.student_id, g.subject_id, sub.name, COALESCE(g.term, '');

-- Attendance per student: percentage = (present + late) / (total - excused) * 100
CREATE VIEW v_student_attendance_summary AS
SELECT a.student_id,
       COUNT(*)                                          AS total_days,
       COUNT(*) FILTER (WHERE a.status = 'present')      AS present_days,
       COUNT(*) FILTER (WHERE a.status = 'late')         AS late_days,
       COUNT(*) FILTER (WHERE a.status = 'absent')       AS absent_days,
       COUNT(*) FILTER (WHERE a.status = 'excused')      AS excused_days,
       COALESCE(ROUND(
           100.0 * COUNT(*) FILTER (WHERE a.status IN ('present', 'late'))
           / NULLIF(COUNT(*) FILTER (WHERE a.status <> 'excused'), 0), 2), 100) AS attendance_percent
FROM attendance a
GROUP BY a.student_id;

-- Class overview: one row per student with name, class, average mark and attendance
CREATE VIEW v_class_overview AS
SELECT c.id                       AS class_id,
       c.name                     AS class_name,
       s.id                       AS student_id,
       s.student_number,
       u.last_name,
       u.first_name,
       (SELECT ROUND(AVG(g.marks), 2) FROM grades g WHERE g.student_id = s.id) AS average_mark,
       COALESCE(att.attendance_percent, 100)                                    AS attendance_percent
FROM students s
JOIN classes c ON c.id = s.class_id
JOIN users u ON u.id = s.user_id
LEFT JOIN v_student_attendance_summary att ON att.student_id = s.id;

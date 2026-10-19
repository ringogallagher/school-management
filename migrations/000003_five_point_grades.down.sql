-- Back to a 0-100 score. The conversion is lossy: each mark becomes the middle of its old band.
ALTER TABLE grades DROP CONSTRAINT IF EXISTS chk_grades_five_point;

UPDATE grades SET marks = CASE
    WHEN marks = 5 THEN 95
    WHEN marks = 4 THEN 82
    WHEN marks = 3 THEN 67
    WHEN marks = 2 THEN 50
    ELSE 20
END;

ALTER TABLE grades ADD CONSTRAINT grades_marks_check CHECK (marks >= 0 AND marks <= 100);

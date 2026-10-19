-- Switch grades from a 0-100 score to the five-point scale (1-5).
-- Existing marks are converted: 90+ -> 5, 75-89 -> 4, 60-74 -> 3, 40-59 -> 2, below 40 -> 1.
UPDATE grades SET marks = CASE
    WHEN marks >= 90 THEN 5
    WHEN marks >= 75 THEN 4
    WHEN marks >= 60 THEN 3
    WHEN marks >= 40 THEN 2
    ELSE 1
END;

ALTER TABLE grades DROP CONSTRAINT IF EXISTS grades_marks_check;
ALTER TABLE grades ADD CONSTRAINT chk_grades_five_point CHECK (marks IN (1, 2, 3, 4, 5));

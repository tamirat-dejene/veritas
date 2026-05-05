ALTER TABLE veritas_questions RENAME COLUMN evaluation_criteria TO metadata;
ALTER TABLE veritas_questions DROP COLUMN expected_answer;
ALTER TABLE veritas_exam_questions ADD COLUMN points_override INT;

ALTER TABLE veritas_exam_questions DROP COLUMN points_override;
ALTER TABLE veritas_questions ADD COLUMN expected_answer TEXT;
ALTER TABLE veritas_questions RENAME COLUMN metadata TO evaluation_criteria;

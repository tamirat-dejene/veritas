-- +migrate Down

DROP INDEX IF EXISTS idx_random_rules_exam;
DROP INDEX IF EXISTS idx_exam_questions_exam;
DROP INDEX IF EXISTS idx_question_options_question;
DROP INDEX IF EXISTS idx_exam_schedule;
DROP INDEX IF EXISTS idx_exams_status;
DROP INDEX IF EXISTS idx_exams_enterprise;
DROP INDEX IF EXISTS idx_questions_enterprise;

DROP TABLE IF EXISTS veritas_exam_randomization_rules;
DROP TABLE IF EXISTS veritas_exam_questions;
DROP TABLE IF EXISTS veritas_exams;
DROP TABLE IF EXISTS veritas_question_options;
DROP TABLE IF EXISTS veritas_questions;

-- Optional: Drop enums if they are not used elsewhere
-- Note: In some systems, enums might be shared, but here they seem specific to exam-service
-- DROP TYPE IF EXISTS exam_status;
-- DROP TYPE IF EXISTS difficulty_level;
-- DROP TYPE IF EXISTS question_type;

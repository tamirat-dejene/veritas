-- +migrate Down

-- Drop indexes first
DROP INDEX IF EXISTS idx_submissions_session;
DROP INDEX IF EXISTS idx_answers_session;
DROP INDEX IF EXISTS idx_sq_session;
DROP INDEX IF EXISTS idx_session_status;
DROP INDEX IF EXISTS idx_session_enrollment;
DROP INDEX IF EXISTS idx_session_candidate;
DROP INDEX IF EXISTS idx_session_exam;
DROP INDEX IF EXISTS idx_enrollment_status;
DROP INDEX IF EXISTS idx_enrollment_candidate;
DROP INDEX IF EXISTS idx_enrollment_exam;
DROP INDEX IF EXISTS idx_candidate_enterprise;

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS exam_submissions;
DROP TABLE IF EXISTS session_answers;
DROP TABLE IF EXISTS session_questions;
DROP TABLE IF EXISTS exam_sessions;
DROP TABLE IF EXISTS exam_enrollments;
DROP TABLE IF EXISTS candidate_profiles;

-- Drop enums
DROP TYPE IF EXISTS session_status;

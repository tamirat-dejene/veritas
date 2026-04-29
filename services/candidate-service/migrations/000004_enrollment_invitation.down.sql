-- Migration 004: Rollback invitation workflow columns
DROP INDEX IF EXISTS idx_enrollment_invite_code;
DROP INDEX IF EXISTS idx_enrollment_status;

ALTER TABLE exam_enrollments
    DROP COLUMN IF EXISTS invitation_sent_at,
    DROP COLUMN IF EXISTS invitation_code_hash,
    DROP COLUMN IF EXISTS status;

-- Drop custom enum type
DROP TYPE IF EXISTS enrollment_status;

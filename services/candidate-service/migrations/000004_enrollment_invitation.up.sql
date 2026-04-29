-- Migration 004: Add invitation workflow columns to exam_enrollments
-- Adds status lifecycle tracking, opaque invitation code (hashed), and invitation timestamp.

-- Create custom enum type for enrollment status
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'enrollment_status') THEN
        CREATE TYPE enrollment_status AS ENUM (
            'Pending',
            'Invited',
            'Opened',
            'Started',
            'Completed',
            'Revoked'
        );
    END IF;
END $$;

ALTER TABLE exam_enrollments
    ADD COLUMN IF NOT EXISTS status               enrollment_status NOT NULL DEFAULT 'Pending',
    ADD COLUMN IF NOT EXISTS invitation_code_hash TEXT              NULL,
    ADD COLUMN IF NOT EXISTS invitation_sent_at   TIMESTAMPTZ       NULL;

-- Index for filtering enrollments by status
CREATE INDEX IF NOT EXISTS idx_enrollment_status
    ON exam_enrollments (status);

-- Index for redeeming invitation codes (candidate click-through)
CREATE UNIQUE INDEX IF NOT EXISTS idx_enrollment_invite_code
    ON exam_enrollments (invitation_code_hash)
    WHERE invitation_code_hash IS NOT NULL;

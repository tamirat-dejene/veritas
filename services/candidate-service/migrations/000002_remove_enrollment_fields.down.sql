ALTER TABLE exam_enrollments ADD COLUMN invitation_method VARCHAR(50) NOT NULL DEFAULT 'Link';
ALTER TABLE exam_enrollments ADD COLUMN status VARCHAR(50) NOT NULL DEFAULT 'Invited';
CREATE INDEX IF NOT EXISTS idx_enrollment_status ON exam_enrollments (status);

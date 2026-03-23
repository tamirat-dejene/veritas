DROP INDEX IF EXISTS idx_enrollment_status;
ALTER TABLE exam_enrollments DROP COLUMN invitation_method;
ALTER TABLE exam_enrollments DROP COLUMN status;

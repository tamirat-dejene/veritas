-- +migrate Down
ALTER TABLE exam_submissions ADD COLUMN total_score NUMERIC(6,2) NULL;
ALTER TABLE exam_submissions ADD COLUMN grading_status VARCHAR(50) NOT NULL DEFAULT 'Pending';

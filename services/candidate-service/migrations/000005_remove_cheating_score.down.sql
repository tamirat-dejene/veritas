-- +migrate Down
ALTER TABLE exam_sessions ADD COLUMN cheating_score NUMERIC(5,2) NULL;

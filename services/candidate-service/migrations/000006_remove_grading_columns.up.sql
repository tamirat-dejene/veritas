-- +migrate Up
ALTER TABLE exam_submissions DROP COLUMN IF EXISTS total_score;
ALTER TABLE exam_submissions DROP COLUMN IF EXISTS grading_status;

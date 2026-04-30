-- +migrate Up
ALTER TABLE exam_sessions DROP COLUMN IF EXISTS cheating_score;

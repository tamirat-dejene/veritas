-- Migration: Add options and correct_option_ids to grading_question_results table
-- Up migration

ALTER TABLE grading_question_results 
ADD COLUMN IF NOT EXISTS options JSONB,
ADD COLUMN IF NOT EXISTS correct_option_ids JSONB;

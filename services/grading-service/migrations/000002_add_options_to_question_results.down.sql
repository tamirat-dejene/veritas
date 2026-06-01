-- Migration: Remove options and correct_option_ids from grading_question_results table
-- Down migration

ALTER TABLE grading_question_results 
DROP COLUMN IF EXISTS options,
DROP COLUMN IF EXISTS correct_option_ids;

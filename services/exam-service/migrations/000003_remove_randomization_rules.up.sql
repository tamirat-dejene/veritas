-- +migrate Up
DROP TABLE IF EXISTS veritas_exam_randomization_rules CASCADE;
DROP INDEX IF EXISTS idx_random_rules_exam;

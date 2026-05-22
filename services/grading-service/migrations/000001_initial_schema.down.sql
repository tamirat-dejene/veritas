-- +migrate Down

DROP TRIGGER IF EXISTS trg_grading_results_audit ON grading_results;
DROP FUNCTION IF EXISTS grading_results_audit_trigger();

DROP TABLE IF EXISTS grading_audit_log;
DROP TABLE IF EXISTS grading_question_results;
DROP TABLE IF EXISTS grading_results;

DROP TYPE IF EXISTS question_grading_status;
DROP TYPE IF EXISTS grading_status;

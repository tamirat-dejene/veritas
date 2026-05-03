-- Remove cross-service FKs that reference tables owned by enterprise-service.
-- enterprise_id and created_by columns are retained as plain UUIDs;
-- referential integrity will be handled via eventual consistency.

ALTER TABLE veritas_questions
    DROP CONSTRAINT IF EXISTS fk_question_enterprise,
    DROP CONSTRAINT IF EXISTS fk_question_creator;

ALTER TABLE veritas_exams
    DROP CONSTRAINT IF EXISTS fk_exam_enterprise,
    DROP CONSTRAINT IF EXISTS fk_exam_creator;

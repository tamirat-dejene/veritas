-- Restore cross-service FKs (for rollback only — not safe in a decoupled setup).

ALTER TABLE veritas_questions
    ADD CONSTRAINT fk_question_enterprise
        FOREIGN KEY (enterprise_id)
        REFERENCES veritas_enterprise(id)
        ON DELETE CASCADE,
    ADD CONSTRAINT fk_question_creator
        FOREIGN KEY (created_by)
        REFERENCES veritas_users(id)
        ON DELETE RESTRICT;

ALTER TABLE veritas_exams
    ADD CONSTRAINT fk_exam_enterprise
        FOREIGN KEY (enterprise_id)
        REFERENCES veritas_enterprise(id)
        ON DELETE CASCADE,
    ADD CONSTRAINT fk_exam_creator
        FOREIGN KEY (created_by)
        REFERENCES veritas_users(id)
        ON DELETE RESTRICT;

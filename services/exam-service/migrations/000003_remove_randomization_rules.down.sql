-- +migrate Down
CREATE TABLE IF NOT EXISTS veritas_exam_randomization_rules (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_id             UUID NOT NULL,

    topic               VARCHAR(255) NULL,
    difficulty          difficulty_level NULL,

    question_count      INT NOT NULL,

    CONSTRAINT fk_random_exam
        FOREIGN KEY (exam_id)
        REFERENCES veritas_exams(id)
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_random_rules_exam
    ON veritas_exam_randomization_rules (exam_id);

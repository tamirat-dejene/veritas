-- +migrate Up

-- 1. Question Bank Enums
DO $$ BEGIN
    CREATE TYPE question_type AS ENUM (
        'MCQ',
        'TrueFalse',
        'ShortAnswer',
        'Essay'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TYPE difficulty_level AS ENUM (
        'Easy',
        'Medium',
        'Hard'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- 2. Questions Table
CREATE TABLE IF NOT EXISTS veritas_questions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enterprise_id       UUID NOT NULL,

    type                question_type NOT NULL,
    topic               VARCHAR(255) NOT NULL,
    difficulty          difficulty_level NOT NULL,

    title               VARCHAR(500) NOT NULL,
    content             TEXT NOT NULL,

    media_url           TEXT NULL,

    points              INT NOT NULL DEFAULT 1,
    negative_points     NUMERIC(5,2) DEFAULT 0,

    metadata            JSONB NOT NULL DEFAULT '{}'::jsonb,

    is_active           BOOLEAN NOT NULL DEFAULT true,

    created_by          UUID NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_question_enterprise
        FOREIGN KEY (enterprise_id)
        REFERENCES veritas_enterprise(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_question_creator
        FOREIGN KEY (created_by)
        REFERENCES veritas_users(id)
        ON DELETE RESTRICT
);

-- 3. MCQ Options Table
CREATE TABLE IF NOT EXISTS veritas_question_options (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id     UUID NOT NULL,
    content         TEXT NOT NULL,
    is_correct      BOOLEAN NOT NULL DEFAULT false,

    CONSTRAINT fk_option_question
        FOREIGN KEY (question_id)
        REFERENCES veritas_questions(id)
        ON DELETE CASCADE
);

-- 4. Exam Status Enum
DO $$ BEGIN
    CREATE TYPE exam_status AS ENUM (
        'Draft',
        'Scheduled',
        'Active',
        'Closed',
        'Archived'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- 5. Exams Table
CREATE TABLE IF NOT EXISTS veritas_exams (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enterprise_id           UUID NOT NULL,

    title                   VARCHAR(255) NOT NULL,
    description             TEXT NULL,

    duration_minutes        INT NOT NULL,
    passing_score_percent   NUMERIC(5,2) NOT NULL,

    negative_marking        BOOLEAN NOT NULL DEFAULT false,

    max_participants        INT NULL,

    invitation_method       VARCHAR(50) NOT NULL, -- Email, Link, Token

    status                  exam_status NOT NULL DEFAULT 'Draft',

    template_source_id      UUID NULL, -- for cloning

    scheduled_start         TIMESTAMPTZ NULL,
    scheduled_end           TIMESTAMPTZ NULL,

    settings                JSONB NOT NULL DEFAULT '{}'::jsonb,

    created_by              UUID NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_exam_enterprise
        FOREIGN KEY (enterprise_id)
        REFERENCES veritas_enterprise(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_exam_creator
        FOREIGN KEY (created_by)
        REFERENCES veritas_users(id)
        ON DELETE RESTRICT,

    CONSTRAINT fk_exam_template
        FOREIGN KEY (template_source_id)
        REFERENCES veritas_exams(id)
        ON DELETE SET NULL
);

-- 6. Exam Question Mapping
CREATE TABLE IF NOT EXISTS veritas_exam_questions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_id         UUID NOT NULL,
    question_id     UUID NOT NULL,
    points_override INT NULL,
    order_index     INT NULL,

    CONSTRAINT fk_exam_question_exam
        FOREIGN KEY (exam_id)
        REFERENCES veritas_exams(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_exam_question_question
        FOREIGN KEY (question_id)
        REFERENCES veritas_questions(id)
        ON DELETE CASCADE,

    CONSTRAINT uq_exam_question
        UNIQUE (exam_id, question_id)
);

-- 7. Targeted Randomization Rules
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

-- 8. Multi-Tenant and Operational Indexing
CREATE INDEX IF NOT EXISTS idx_questions_enterprise
    ON veritas_questions (enterprise_id);

CREATE INDEX IF NOT EXISTS idx_exams_enterprise
    ON veritas_exams (enterprise_id);

CREATE INDEX IF NOT EXISTS idx_exams_status
    ON veritas_exams (status);

CREATE INDEX IF NOT EXISTS idx_exam_schedule
    ON veritas_exams (scheduled_start, scheduled_end);

CREATE INDEX IF NOT EXISTS idx_question_options_question
    ON veritas_question_options (question_id);

CREATE INDEX IF NOT EXISTS idx_exam_questions_exam
    ON veritas_exam_questions (exam_id);

CREATE INDEX IF NOT EXISTS idx_random_rules_exam
    ON veritas_exam_randomization_rules (exam_id);

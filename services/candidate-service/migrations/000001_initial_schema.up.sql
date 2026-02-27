-- +migrate Up

-- =========================================================
-- ENUMS
-- =========================================================

DO $$ BEGIN
    CREATE TYPE session_status AS ENUM (
        'Active',
        'Submitted',
        'Terminated',
        'Expired'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- =========================================================
-- 1. Candidate Profiles
--    Core identity for candidates (no email/password auth).
--    Scoped to enterprise; supports bulk CSV upload.
-- =========================================================
CREATE TABLE IF NOT EXISTS candidate_profiles (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enterprise_id       UUID NOT NULL,

    external_id         VARCHAR(255) NOT NULL, -- from CSV upload
    first_name          VARCHAR(255) NOT NULL,
    last_name           VARCHAR(255) NOT NULL,
    email               VARCHAR(255) NULL,

    face_reference_url  TEXT NULL, -- FR-AUTH-004

    is_active           BOOLEAN NOT NULL DEFAULT true,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_candidate_external
        UNIQUE (enterprise_id, external_id)
);

-- =========================================================
-- 2. Exam Enrollments
--    Links a candidate to an exam and issues a secure token.
--    Token is stored hashed — never raw.
-- =========================================================
CREATE TABLE IF NOT EXISTS exam_enrollments (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enterprise_id           UUID NOT NULL,
    exam_id                 UUID NOT NULL,
    candidate_id            UUID NOT NULL,

    invitation_method       VARCHAR(50) NOT NULL, -- Email, Link, Token
    access_token_hash       TEXT NOT NULL,
    token_expires_at        TIMESTAMPTZ NOT NULL,

    max_attempts            INT NOT NULL DEFAULT 1,
    attempts_used           INT NOT NULL DEFAULT 0,

    status                  VARCHAR(50) NOT NULL DEFAULT 'Invited',

    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_enrollment_candidate
        FOREIGN KEY (candidate_id)
        REFERENCES candidate_profiles(id)
        ON DELETE CASCADE,

    CONSTRAINT uq_candidate_exam
        UNIQUE (exam_id, candidate_id)
);

-- =========================================================
-- 3. Exam Sessions
--    Heart of the system. One session per attempt.
--    Supports server-enforced timer, termination, and
--    cheating score tracking.
-- =========================================================
CREATE TABLE IF NOT EXISTS exam_sessions (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enterprise_id           UUID NOT NULL,
    exam_id                 UUID NOT NULL,
    candidate_id            UUID NOT NULL,
    enrollment_id           UUID NOT NULL,

    status                  session_status NOT NULL DEFAULT 'Active',

    started_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at              TIMESTAMPTZ NOT NULL,   -- server-enforced timer
    submitted_at            TIMESTAMPTZ NULL,
    terminated_at           TIMESTAMPTZ NULL,

    termination_reason      TEXT NULL,

    client_ip               INET NULL,
    user_agent              TEXT NULL,

    face_registered_url     TEXT NULL, -- onboarding face image

    cheating_score          NUMERIC(5,2) NULL,

    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_session_candidate
        FOREIGN KEY (candidate_id)
        REFERENCES candidate_profiles(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_session_enrollment
        FOREIGN KEY (enrollment_id)
        REFERENCES exam_enrollments(id)
        ON DELETE CASCADE
);

-- =========================================================
-- 4. Session Questions Snapshot
--    Critical: Snapshot selected questions at session start
--    so that question edits or cloning never affect an
--    active or past session.
-- =========================================================
CREATE TABLE IF NOT EXISTS session_questions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id          UUID NOT NULL,

    question_id         UUID NOT NULL, -- reference to exam-service question
    question_snapshot   JSONB NOT NULL, -- full immutable copy at time of session

    order_index         INT NOT NULL,

    points              INT NOT NULL,
    negative_points     NUMERIC(5,2) NOT NULL DEFAULT 0,

    CONSTRAINT fk_sq_session
        FOREIGN KEY (session_id)
        REFERENCES exam_sessions(id)
        ON DELETE CASCADE
);

-- =========================================================
-- 5. Session Answers
--    Supports auto-save (every 60s), partial save, and
--    offline sync. One row per question per session
--    (upserted on each save). Marked final on submission.
-- =========================================================
CREATE TABLE IF NOT EXISTS session_answers (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id              UUID NOT NULL,
    session_question_id     UUID NOT NULL,

    -- Flexible: MCQ = array, Essay = text, TrueFalse = boolean
    answer_data             JSONB NOT NULL,
    is_final                BOOLEAN NOT NULL DEFAULT false,

    saved_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_answer_session
        FOREIGN KEY (session_id)
        REFERENCES exam_sessions(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_answer_sq
        FOREIGN KEY (session_question_id)
        REFERENCES session_questions(id)
        ON DELETE CASCADE,

    CONSTRAINT uq_session_question
        UNIQUE (session_id, session_question_id)
);

-- =========================================================
-- 6. Exam Submissions
--    Created once per session on submit (manual or auto).
--    Grading status tracks the lifecycle for AI/manual
--    grading integration.
-- =========================================================
CREATE TABLE IF NOT EXISTS exam_submissions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id          UUID NOT NULL UNIQUE,

    submitted_at        TIMESTAMPTZ NOT NULL,
    auto_submitted      BOOLEAN NOT NULL DEFAULT false,

    total_score         NUMERIC(6,2) NULL,
    grading_status      VARCHAR(50) NOT NULL DEFAULT 'Pending',

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_submission_session
        FOREIGN KEY (session_id)
        REFERENCES exam_sessions(id)
        ON DELETE CASCADE
);

-- =========================================================
-- INDEXES — Multi-Tenant & Operational
-- =========================================================

-- Candidate Profiles
CREATE INDEX IF NOT EXISTS idx_candidate_enterprise
    ON candidate_profiles (enterprise_id);

-- Enrollments
CREATE INDEX IF NOT EXISTS idx_enrollment_exam
    ON exam_enrollments (exam_id);

CREATE INDEX IF NOT EXISTS idx_enrollment_candidate
    ON exam_enrollments (candidate_id);

CREATE INDEX IF NOT EXISTS idx_enrollment_status
    ON exam_enrollments (status);

-- Sessions
CREATE INDEX IF NOT EXISTS idx_session_exam
    ON exam_sessions (exam_id);

CREATE INDEX IF NOT EXISTS idx_session_candidate
    ON exam_sessions (candidate_id);

CREATE INDEX IF NOT EXISTS idx_session_enrollment
    ON exam_sessions (enrollment_id);

CREATE INDEX IF NOT EXISTS idx_session_status
    ON exam_sessions (status);

-- Session Questions
CREATE INDEX IF NOT EXISTS idx_sq_session
    ON session_questions (session_id);

-- Answers
CREATE INDEX IF NOT EXISTS idx_answers_session
    ON session_answers (session_id);

-- Submissions
CREATE INDEX IF NOT EXISTS idx_submissions_session
    ON exam_submissions (session_id);

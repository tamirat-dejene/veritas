-- +migrate Up

-- =========================================================
-- ENUMS
-- =========================================================

DO $$ BEGIN
    CREATE TYPE grading_status AS ENUM ('pending', 'graded', 'reviewed', 'disputed');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TYPE question_grading_status AS ENUM ('correct', 'incorrect', 'partial', 'skipped', 'ai_graded');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- =========================================================
-- 1. Exam Grade Results (one row per graded session)
--
-- Security features:
--   • row_checksum — SHA-256 over immutable scoring fields;
--     any direct UPDATE to score columns without updating the
--     checksum will be caught by the application layer.
--   • graded_by — tracks whether the system or a human reviewed.
--   • version — optimistic-concurrency control; every legitimate
--     update bumps the version. Stale writes are rejected.
-- =========================================================

CREATE TABLE IF NOT EXISTS grading_results (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id          UUID NOT NULL UNIQUE,
    exam_id             UUID NOT NULL,
    candidate_id        UUID NOT NULL,
    enterprise_id       UUID NOT NULL,
    enrollment_id       UUID NOT NULL,

    -- Scoring
    total_max_points    NUMERIC(8,2)  NOT NULL DEFAULT 0,
    total_awarded_points NUMERIC(8,2) NOT NULL DEFAULT 0,
    percentage          NUMERIC(5,2)  NOT NULL DEFAULT 0,

    status              grading_status NOT NULL DEFAULT 'graded',
    graded_by           VARCHAR(100) NOT NULL DEFAULT 'system',

    -- Integrity
    row_checksum        VARCHAR(64) NOT NULL,
    version             INT NOT NULL DEFAULT 1,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_grading_results_exam
    ON grading_results (exam_id);

CREATE INDEX IF NOT EXISTS idx_grading_results_candidate
    ON grading_results (candidate_id);

CREATE INDEX IF NOT EXISTS idx_grading_results_enterprise
    ON grading_results (enterprise_id);

CREATE INDEX IF NOT EXISTS idx_grading_results_session
    ON grading_results (session_id);

CREATE INDEX IF NOT EXISTS idx_grading_results_enterprise_exam
    ON grading_results (enterprise_id, exam_id);

-- =========================================================
-- 2. Question-Level Grade Details
--    One row per question in a graded session.
-- =========================================================

CREATE TABLE IF NOT EXISTS grading_question_results (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    grading_result_id    UUID NOT NULL REFERENCES grading_results(id) ON DELETE CASCADE,
    question_id          UUID NOT NULL,
    session_question_id  UUID NOT NULL,
    question_type        VARCHAR(30) NOT NULL,
    title                VARCHAR(500) NOT NULL DEFAULT '',
    max_points           NUMERIC(8,2) NOT NULL DEFAULT 0,
    awarded_points       NUMERIC(8,2) NOT NULL DEFAULT 0,
    status               question_grading_status NOT NULL DEFAULT 'skipped',
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_grading_qr_result
    ON grading_question_results (grading_result_id);

CREATE INDEX IF NOT EXISTS idx_grading_qr_question
    ON grading_question_results (question_id);

-- =========================================================
-- 3. Grade Audit Log (tamper detection)
--
-- Every UPDATE to grading_results is captured here
-- automatically by a trigger. This table is APPEND-ONLY;
-- the application layer must never DELETE or UPDATE rows.
--
-- Stores a JSON snapshot of the OLD and NEW row values,
-- who made the change, and a timestamp.
-- =========================================================

CREATE TABLE IF NOT EXISTS grading_audit_log (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    grading_result_id UUID NOT NULL,
    action          VARCHAR(20) NOT NULL,       -- 'INSERT' | 'UPDATE' | 'DELETE'
    actor_id        UUID,                        -- who made the change (NULL = system)
    actor_role      VARCHAR(50) DEFAULT 'system',
    old_values      JSONB,
    new_values      JSONB NOT NULL,
    changed_fields  TEXT[],                      -- list of column names that changed
    ip_address      VARCHAR(45),
    reason          TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_grading_audit_result
    ON grading_audit_log (grading_result_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_grading_audit_actor
    ON grading_audit_log (actor_id);

CREATE INDEX IF NOT EXISTS idx_grading_audit_action
    ON grading_audit_log (action);

-- =========================================================
-- 4. Audit trigger — auto-log every INSERT/UPDATE/DELETE
--    on grading_results.
-- =========================================================

CREATE OR REPLACE FUNCTION grading_results_audit_trigger()
RETURNS TRIGGER AS $$
DECLARE
    changed TEXT[] := '{}';
    col TEXT;
BEGIN
    IF TG_OP = 'UPDATE' THEN
        -- Detect which columns actually changed
        IF OLD.total_max_points IS DISTINCT FROM NEW.total_max_points THEN
            changed := array_append(changed, 'total_max_points');
        END IF;
        IF OLD.total_awarded_points IS DISTINCT FROM NEW.total_awarded_points THEN
            changed := array_append(changed, 'total_awarded_points');
        END IF;
        IF OLD.percentage IS DISTINCT FROM NEW.percentage THEN
            changed := array_append(changed, 'percentage');
        END IF;
        IF OLD.status IS DISTINCT FROM NEW.status THEN
            changed := array_append(changed, 'status');
        END IF;
        IF OLD.row_checksum IS DISTINCT FROM NEW.row_checksum THEN
            changed := array_append(changed, 'row_checksum');
        END IF;
        IF OLD.graded_by IS DISTINCT FROM NEW.graded_by THEN
            changed := array_append(changed, 'graded_by');
        END IF;

        INSERT INTO grading_audit_log (
            grading_result_id, action, old_values, new_values, changed_fields
        ) VALUES (
            NEW.id,
            'UPDATE',
            to_jsonb(OLD),
            to_jsonb(NEW),
            changed
        );

        -- Enforce version bump
        NEW.version := OLD.version + 1;
        NEW.updated_at := NOW();

        RETURN NEW;

    ELSIF TG_OP = 'INSERT' THEN
        INSERT INTO grading_audit_log (
            grading_result_id, action, new_values
        ) VALUES (
            NEW.id,
            'INSERT',
            to_jsonb(NEW)
        );
        RETURN NEW;

    ELSIF TG_OP = 'DELETE' THEN
        INSERT INTO grading_audit_log (
            grading_result_id, action, old_values, new_values
        ) VALUES (
            OLD.id,
            'DELETE',
            to_jsonb(OLD),
            '{}'::jsonb
        );
        RETURN OLD;
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_grading_results_audit ON grading_results;
CREATE TRIGGER trg_grading_results_audit
    BEFORE INSERT OR UPDATE OR DELETE ON grading_results
    FOR EACH ROW EXECUTE FUNCTION grading_results_audit_trigger();

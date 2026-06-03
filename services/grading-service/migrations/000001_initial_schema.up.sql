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
    CREATE TYPE question_grading_status AS ENUM ('correct', 'incorrect', 'partial', 'skipped', 'ai_graded', 'human_review');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- =========================================================
-- 1. Exam Grade Results (one row per graded session)
--
-- Security features:
--   • row_checksum — HMAC-SHA256 checksum over immutable scoring fields;
--     any direct UPDATE to score columns without updating the
--     checksum will be caught by the application layer.
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
    content              TEXT NOT NULL DEFAULT '',
    candidate_answer     JSONB,
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
-- Every INSERT/UPDATE/DELETE to grading_results is captured here
-- automatically by a trigger. This table is APPEND-ONLY.
--
-- Stores session parameters set by the application during the transaction.
-- =========================================================

CREATE TABLE IF NOT EXISTS grading_audit_log (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    grading_result_id UUID NOT NULL,
    action          VARCHAR(20) NOT NULL,       -- 'INSERT' | 'UPDATE' | 'DELETE'
    actor_id        UUID,                        -- who made the change
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

-- =========================================================
-- 4. Audit trigger — auto-log every INSERT/UPDATE/DELETE
--    on grading_results.
-- =========================================================

CREATE OR REPLACE FUNCTION grading_results_audit_trigger()
RETURNS TRIGGER AS $$
DECLARE
    changed TEXT[] := '{}';
    actor_id_val UUID := NULL;
    actor_role_val VARCHAR(50) := 'system';
    ip_address_val VARCHAR(45) := NULL;
    reason_val TEXT := NULL;
    session_actor_id TEXT;
    session_actor_role TEXT;
    session_ip TEXT;
    session_reason TEXT;
BEGIN
    -- Read session parameters if available
    session_actor_id := current_setting('veritas.current_actor_id', true);
    session_actor_role := current_setting('veritas.current_actor_role', true);
    session_ip := current_setting('veritas.current_ip', true);
    session_reason := current_setting('veritas.current_reason', true);

    IF session_actor_id IS NOT NULL AND session_actor_id <> '' THEN
        actor_id_val := session_actor_id::UUID;
    END IF;
    IF session_actor_role IS NOT NULL AND session_actor_role <> '' THEN
        actor_role_val := session_actor_role;
    END IF;
    IF session_ip IS NOT NULL AND session_ip <> '' THEN
        ip_address_val := session_ip;
    END IF;
    IF session_reason IS NOT NULL AND session_reason <> '' THEN
        reason_val := session_reason;
    END IF;

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
            grading_result_id, action, actor_id, actor_role, old_values, new_values, changed_fields, ip_address, reason
        ) VALUES (
            NEW.id,
            'UPDATE',
            actor_id_val,
            actor_role_val,
            to_jsonb(OLD),
            to_jsonb(NEW),
            changed,
            ip_address_val,
            reason_val
        );

        -- Enforce version bump
        NEW.version := OLD.version + 1;
        NEW.updated_at := NOW();

        RETURN NEW;

    ELSIF TG_OP = 'INSERT' THEN
        INSERT INTO grading_audit_log (
            grading_result_id, action, actor_id, actor_role, old_values, new_values, ip_address, reason
        ) VALUES (
            NEW.id,
            'INSERT',
            actor_id_val,
            actor_role_val,
            NULL,
            to_jsonb(NEW),
            ip_address_val,
            reason_val
        );
        RETURN NEW;

    ELSIF TG_OP = 'DELETE' THEN
        INSERT INTO grading_audit_log (
            grading_result_id, action, actor_id, actor_role, old_values, new_values, ip_address, reason
        ) VALUES (
            OLD.id,
            'DELETE',
            actor_id_val,
            actor_role_val,
            to_jsonb(OLD),
            '{}'::jsonb,
            ip_address_val,
            reason_val
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

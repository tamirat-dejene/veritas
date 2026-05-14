-- +migrate Up

-- =========================================================
-- ENUMS
-- =========================================================

DO $$ BEGIN
    CREATE TYPE proctoring_severity AS ENUM ('low', 'medium', 'high', 'critical');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- =========================================================
-- 1. Proctoring Events
--    Every behavioral signal detected during an exam session.
--    No screenshots stored — structured metadata only.
--    Event types: tab_switch, mouse_inactive, face_not_detected,
--    multiple_faces, identity_mismatch, copy_paste_attempt,
--    fullscreen_exit, periodic_face_ok
-- =========================================================

CREATE TABLE IF NOT EXISTS proctoring_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id      UUID NOT NULL,
    candidate_id    UUID NOT NULL,
    enterprise_id   UUID NOT NULL,
    event_type      VARCHAR(50) NOT NULL,
    severity        proctoring_severity NOT NULL DEFAULT 'low',
    metadata        JSONB NOT NULL DEFAULT '{}',
    occurred_at     TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_proc_events_session
    ON proctoring_events (session_id);

CREATE INDEX IF NOT EXISTS idx_proc_events_enterprise
    ON proctoring_events (enterprise_id);

CREATE INDEX IF NOT EXISTS idx_proc_events_type
    ON proctoring_events (event_type);

CREATE INDEX IF NOT EXISTS idx_proc_events_occurred
    ON proctoring_events (occurred_at DESC);

-- =========================================================
-- 2. Session Cheating Score
--    Upserted on every new behavioral event.
--    cheating_score is additive-weighted, capped at 100.
--    is_final is set externally (by consumer of
--    candidate.exam.submitted) — not stored here, it is only
--    in the Kafka payload.
-- =========================================================

CREATE TABLE IF NOT EXISTS proctoring_session_scores (
    session_id          UUID PRIMARY KEY,
    candidate_id        UUID NOT NULL,
    enterprise_id       UUID NOT NULL,
    cheating_score      NUMERIC(5,2) NOT NULL DEFAULT 0,
    event_count         INT NOT NULL DEFAULT 0,
    last_computed_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_scores_enterprise
    ON proctoring_session_scores (enterprise_id);

CREATE INDEX IF NOT EXISTS idx_scores_score
    ON proctoring_session_scores (cheating_score DESC);

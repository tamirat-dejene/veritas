-- +migrate Up

CREATE TABLE IF NOT EXISTS veritas_enterprise_audit_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enterprise_id   UUID NOT NULL REFERENCES veritas_enterprise(id) ON DELETE CASCADE,
    actor_id        UUID NOT NULL,
    actor_role      VARCHAR(50) NOT NULL,
    event           VARCHAR(100) NOT NULL,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_enterprise_created
    ON veritas_enterprise_audit_logs (enterprise_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_event
    ON veritas_enterprise_audit_logs (event);

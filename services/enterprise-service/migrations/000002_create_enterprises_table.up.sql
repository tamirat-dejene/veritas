-- +migrate Up

CREATE TYPE enterprise_status AS ENUM (
    'PendingApproval',
    'Active',
    'Suspended',
    'Deleted'
);

CREATE TYPE subscription_status AS ENUM (
    'Trial',
    'Active',
    'PastDue',
    'Canceled',
    'Expired'
);

CREATE TABLE IF NOT EXISTS veritas_enterprise (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug                    VARCHAR(120) NOT NULL UNIQUE,

    display_name            VARCHAR(255) NOT NULL,
    legal_name              VARCHAR(255) NOT NULL,
    contact_email           VARCHAR(255) NOT NULL,

    owner_account_id        UUID NOT NULL,

    status                  enterprise_status NOT NULL DEFAULT 'PendingApproval',
    approved_at             TIMESTAMP WITH TIME ZONE NULL,
    suspended_at            TIMESTAMP WITH TIME ZONE NULL,
    deleted_at              TIMESTAMP WITH TIME ZONE NULL,
    retention_until         TIMESTAMP WITH TIME ZONE NULL,

    subscription_plan_id    UUID NULL,
    subscription_status     subscription_status NULL,
    current_period_start    TIMESTAMP WITH TIME ZONE NULL,
    current_period_end      TIMESTAMP WITH TIME ZONE NULL,

    logo_url                TEXT NULL,
    primary_color           VARCHAR(7) NULL,   -- HEX format (#RRGGBB)
    secondary_color         VARCHAR(7) NULL,
    custom_domain           VARCHAR(255) NULL UNIQUE,

    contact_phone           VARCHAR(50) NULL,
    address_line1           VARCHAR(255) NULL,
    address_line2           VARCHAR(255) NULL,
    city                    VARCHAR(100) NULL,
    country                 VARCHAR(100) NULL,

    settings                JSONB NOT NULL DEFAULT '{}'::jsonb,

    created_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_by              UUID NOT NULL,
    updated_by              UUID NOT NULL,

    CONSTRAINT fk_enterprise_owner
        FOREIGN KEY (owner_account_id)
        REFERENCES veritas_users(id)
        ON DELETE RESTRICT,

    CONSTRAINT chk_slug_format
        CHECK (slug ~ '^[a-z0-9-]+$')
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_enterprises_status
    ON veritas_enterprise (status);

CREATE INDEX IF NOT EXISTS idx_enterprises_owner
    ON veritas_enterprise (owner_account_id);

CREATE INDEX IF NOT EXISTS idx_enterprises_subscription_status
    ON veritas_enterprise (subscription_status);

CREATE INDEX IF NOT EXISTS idx_enterprises_active
    ON veritas_enterprise (status)
    WHERE status = 'Active';

CREATE INDEX IF NOT EXISTS idx_enterprises_subscription_expiry
    ON veritas_enterprise (current_period_end)
    WHERE subscription_status = 'Active';

CREATE INDEX IF NOT EXISTS idx_enterprises_settings
    ON veritas_enterprise
    USING GIN (settings);

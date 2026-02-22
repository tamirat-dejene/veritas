-- +migrate Up
CREATE TABLE IF NOT EXISTS veritas_users (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email                VARCHAR(255) UNIQUE NOT NULL,
    password_hash        TEXT NOT NULL,
    
    honorific            VARCHAR(50),
    first_name           VARCHAR(255),
    last_name            VARCHAR(255),
    phone                VARCHAR(50) NULL,

    role                 VARCHAR(50) NOT NULL,
    enterprise_id        UUID NULL,

    is_active            BOOLEAN NOT NULL DEFAULT true,
    is_deleted           BOOLEAN NOT NULL DEFAULT false,

    email_verified       BOOLEAN NOT NULL DEFAULT false,
    email_verified_at    TIMESTAMP WITH TIME ZONE NULL,

    failed_login_attempts INT NOT NULL DEFAULT 0,
    locked_until         TIMESTAMP WITH TIME ZONE NULL,

    password_changed_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    must_change_password BOOLEAN NOT NULL DEFAULT false,

    last_login_at        TIMESTAMP WITH TIME ZONE NULL,
    last_login_ip        INET NULL,
    last_user_agent      TEXT NULL,

    created_at           TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_veritas_users_email ON veritas_users (email);
CREATE INDEX IF NOT EXISTS idx_veritas_users_role ON veritas_users (role);
CREATE INDEX IF NOT EXISTS idx_veritas_users_enterprise_id ON veritas_users (enterprise_id) WHERE enterprise_id IS NOT NULL;

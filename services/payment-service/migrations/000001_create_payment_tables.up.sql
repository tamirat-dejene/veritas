-- +migrate Up

-- 0. Define Enums
DO $$ BEGIN
    CREATE TYPE billing_cycle_type AS ENUM ('monthly', 'yearly');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TYPE currency_type AS ENUM ('ETB', 'USD', 'EUR', 'GBP');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TYPE sub_status_type AS ENUM ('Active', 'PastDue', 'Canceled', 'Expired', 'Trial');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TYPE invoice_status_type AS ENUM ('Draft', 'Open', 'Paid', 'Void', 'Uncollectible');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TYPE payment_status_type AS ENUM ('Pending', 'Succeeded', 'Failed', 'Refunded');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- 1. Subscription Plans
CREATE TABLE IF NOT EXISTS veritas_subscription_plans (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(100) NOT NULL UNIQUE,
    slug            VARCHAR(100) NOT NULL UNIQUE,
    description     TEXT,
    
    price           DECIMAL(12, 2) NOT NULL DEFAULT 0.00,
    currency        currency_type NOT NULL DEFAULT 'ETB',
    billing_cycle   billing_cycle_type NOT NULL DEFAULT 'monthly',
    
    -- Feature Limits (stored as JSON for flexibility)
    features        JSONB NOT NULL DEFAULT '{}'::jsonb,
    
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 2. Enterprise Subscriptions
CREATE TABLE IF NOT EXISTS veritas_enterprise_subscriptions (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enterprise_id           UUID NOT NULL UNIQUE,
    plan_id                 UUID NOT NULL,
    
    status                  sub_status_type NOT NULL DEFAULT 'Active',
    
    current_period_start    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    current_period_end      TIMESTAMP WITH TIME ZONE NOT NULL,
    
    cancel_at_period_end    BOOLEAN NOT NULL DEFAULT false,
    canceled_at             TIMESTAMP WITH TIME ZONE NULL,
    ended_at                TIMESTAMP WITH TIME ZONE NULL,
    
    -- External Provider Refs
    stripe_customer_id      VARCHAR(255) NULL,
    stripe_subscription_id  VARCHAR(255) NULL,
    
    created_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT fk_subscription_plan
        FOREIGN KEY (plan_id)
        REFERENCES veritas_subscription_plans(id)
        ON DELETE RESTRICT
);

-- 3. Invoices
CREATE TABLE IF NOT EXISTS veritas_invoices (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enterprise_id           UUID NOT NULL,
    subscription_id         UUID NOT NULL,
    
    number                  VARCHAR(50) NOT NULL UNIQUE, -- INV-XXXXXX
    status                  invoice_status_type NOT NULL DEFAULT 'Draft',
    
    amount_due              DECIMAL(12, 2) NOT NULL,
    amount_paid             DECIMAL(12, 2) NOT NULL DEFAULT 0.00,
    amount_remaining        DECIMAL(12, 2) NOT NULL,
    currency                currency_type NOT NULL DEFAULT 'ETB',
    
    due_date                TIMESTAMP WITH TIME ZONE NOT NULL,
    paid_at                 TIMESTAMP WITH TIME ZONE NULL,
    
    hosted_invoice_url      TEXT NULL,
    invoice_pdf_url        TEXT NULL,
    
    created_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT fk_invoice_enterprise
        FOREIGN KEY (enterprise_id)
        REFERENCES veritas_enterprise_subscriptions(enterprise_id)
        ON DELETE CASCADE,
    CONSTRAINT fk_invoice_subscription
        FOREIGN KEY (subscription_id)
        REFERENCES veritas_enterprise_subscriptions(id)
        ON DELETE CASCADE
);

-- 4. Payments (Transactions)
CREATE TABLE IF NOT EXISTS veritas_payments (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enterprise_id           UUID NOT NULL,
    invoice_id              UUID NULL,
    
    amount                  DECIMAL(12, 2) NOT NULL,
    currency                currency_type NOT NULL DEFAULT 'ETB',
    
    status                  payment_status_type NOT NULL,
    payment_method_type     VARCHAR(50) NULL, -- credit_card, bank_transfer, etc.
    
    -- Provider Refs
    provider                VARCHAR(50) NOT NULL, -- stripe, paypal
    provider_payment_id     VARCHAR(255) NOT NULL UNIQUE,
    provider_error_code     VARCHAR(100) NULL,
    provider_error_message  TEXT NULL,
    
    created_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT fk_payment_enterprise
        FOREIGN KEY (enterprise_id)
        REFERENCES veritas_enterprise_subscriptions(enterprise_id)
        ON DELETE CASCADE,
    CONSTRAINT fk_payment_invoice
        FOREIGN KEY (invoice_id)
        REFERENCES veritas_invoices(id)
        ON DELETE SET NULL
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_subs_enterprise ON veritas_enterprise_subscriptions(enterprise_id);
CREATE INDEX IF NOT EXISTS idx_subs_status ON veritas_enterprise_subscriptions(status);
CREATE INDEX IF NOT EXISTS idx_invoices_enterprise ON veritas_invoices(enterprise_id);
CREATE INDEX IF NOT EXISTS idx_invoices_status ON veritas_invoices(status);
CREATE INDEX IF NOT EXISTS idx_payments_enterprise ON veritas_payments(enterprise_id);
CREATE INDEX IF NOT EXISTS idx_payments_status ON veritas_payments(status);
CREATE INDEX IF NOT EXISTS idx_plans_slug ON veritas_subscription_plans(slug);

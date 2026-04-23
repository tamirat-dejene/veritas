-- +migrate Up

-- Drop indexes that reference the removed columns
DROP INDEX IF EXISTS idx_enterprises_subscription_status;
DROP INDEX IF EXISTS idx_enterprises_subscription_expiry;

-- Drop subscription columns from veritas_enterprise.
-- Subscription state is now owned exclusively by the payment-service.
ALTER TABLE veritas_enterprise
    DROP COLUMN IF EXISTS subscription_plan_id,
    DROP COLUMN IF EXISTS subscription_status,
    DROP COLUMN IF EXISTS current_period_start,
    DROP COLUMN IF EXISTS current_period_end;

-- Drop the now-unused enum type (safe to drop after columns are gone)
DROP TYPE IF EXISTS subscription_status;

-- +migrate Down

-- Re-create the subscription_status enum type
CREATE TYPE subscription_status AS ENUM (
    'Trial',
    'Active',
    'PastDue',
    'Canceled',
    'Expired'
);

-- Re-add subscription columns to veritas_enterprise
ALTER TABLE veritas_enterprise
    ADD COLUMN subscription_plan_id    UUID NULL,
    ADD COLUMN subscription_status     subscription_status NULL,
    ADD COLUMN current_period_start    TIMESTAMP WITH TIME ZONE NULL,
    ADD COLUMN current_period_end      TIMESTAMP WITH TIME ZONE NULL;

-- Re-create indexes
CREATE INDEX idx_enterprises_subscription_status
    ON veritas_enterprise (subscription_status);

CREATE INDEX idx_enterprises_subscription_expiry
    ON veritas_enterprise (current_period_end)
    WHERE subscription_status = 'Active';

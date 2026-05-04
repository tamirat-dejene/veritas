-- +migrate Down
ALTER TYPE billing_cycle_type ADD VALUE IF NOT EXISTS 'month';
ALTER TYPE billing_cycle_type ADD VALUE IF NOT EXISTS 'year';

UPDATE veritas_subscription_plans SET billing_cycle = 'monthly' WHERE billing_cycle::TEXT = 'month';
UPDATE veritas_subscription_plans SET billing_cycle = 'yearly' WHERE billing_cycle::TEXT = 'year';

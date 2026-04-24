-- +migrate Up
ALTER TABLE veritas_subscription_plans ADD COLUMN stripe_price_id VARCHAR(100);

-- +migrate Down
ALTER TABLE veritas_subscription_plans DROP COLUMN stripe_price_id;

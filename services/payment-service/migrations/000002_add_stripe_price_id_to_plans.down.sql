-- +migrate Down
ALTER TABLE veritas_subscription_plans DROP COLUMN stripe_price_id;

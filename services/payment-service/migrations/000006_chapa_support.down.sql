-- Drop columns and indexes added for Chapa support
DROP INDEX IF EXISTS idx_subs_chapa_tx_ref;

ALTER TABLE veritas_enterprise_subscriptions
    DROP COLUMN IF EXISTS chapa_tx_ref,
    DROP COLUMN IF EXISTS payment_provider;

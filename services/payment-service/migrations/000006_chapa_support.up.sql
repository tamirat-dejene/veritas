-- Add columns to support Chapa integration parallel to Stripe
ALTER TABLE veritas_enterprise_subscriptions
    ADD COLUMN IF NOT EXISTS chapa_tx_ref VARCHAR(255) NULL,
    ADD COLUMN IF NOT EXISTS payment_provider VARCHAR(50) NOT NULL DEFAULT 'stripe';

-- Create index for Chapa tx_ref lookups
CREATE INDEX IF NOT EXISTS idx_subs_chapa_tx_ref ON veritas_enterprise_subscriptions(chapa_tx_ref)
    WHERE chapa_tx_ref IS NOT NULL;

-- +migrate Down

DROP TABLE IF EXISTS veritas_payments;
DROP TABLE IF EXISTS veritas_invoices;
DROP TABLE IF EXISTS veritas_enterprise_subscriptions;
DROP TABLE IF EXISTS veritas_subscription_plans;

DROP TYPE IF EXISTS payment_status_type;
DROP TYPE IF EXISTS invoice_status_type;
DROP TYPE IF EXISTS sub_status_type;
DROP TYPE IF EXISTS currency_type;
DROP TYPE IF EXISTS billing_cycle_type;

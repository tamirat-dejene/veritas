-- +migrate Down

DROP TABLE IF EXISTS veritas_enterprise;
DROP TYPE IF EXISTS subscription_status;
DROP TYPE IF EXISTS enterprise_status;

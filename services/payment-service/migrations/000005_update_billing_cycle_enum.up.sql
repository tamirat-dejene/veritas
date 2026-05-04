-- +migrate Up

-- 0. Add new values to the enum
ALTER TYPE billing_cycle_type ADD VALUE IF NOT EXISTS 'month';
ALTER TYPE billing_cycle_type ADD VALUE IF NOT EXISTS 'year';

-- 1. Update existing data to use the new values
UPDATE veritas_subscription_plans SET billing_cycle = 'month' WHERE billing_cycle = 'monthly';
UPDATE veritas_subscription_plans SET billing_cycle = 'year' WHERE billing_cycle = 'yearly';

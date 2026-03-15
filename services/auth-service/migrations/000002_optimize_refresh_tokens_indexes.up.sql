-- +migrate Up
DROP INDEX IF EXISTS idx_refresh_tokens_token_hash;
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id_expires_at ON refresh_tokens (user_id, expires_at);

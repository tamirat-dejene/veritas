-- +migrate Down
DROP INDEX IF EXISTS idx_refresh_tokens_user_id_expires_at;
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token_hash ON refresh_tokens (token_hash);

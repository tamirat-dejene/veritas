-- Restore the FK (for rollback only — not safe in a fully decoupled setup).
ALTER TABLE refresh_tokens
    ADD CONSTRAINT refresh_tokens_user_id_fkey
        FOREIGN KEY (user_id)
        REFERENCES veritas_users(id)
        ON DELETE CASCADE;

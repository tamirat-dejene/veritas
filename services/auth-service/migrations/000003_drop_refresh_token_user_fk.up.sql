-- Remove the cross-service FK that references veritas_users(id) in enterprise-service.
-- user_id is retained as a plain UUID; referential integrity will be handled via
-- eventual consistency once that pattern is implemented.
ALTER TABLE refresh_tokens
    DROP CONSTRAINT IF EXISTS refresh_tokens_user_id_fkey;

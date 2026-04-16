-- +migrate Down
DELETE FROM veritas_users WHERE email = 'admin@veritas.com';

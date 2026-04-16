-- +migrate Up
INSERT INTO veritas_users (
    id,
    email,
    password_hash,
    first_name,
    last_name,
    role,
    is_active,
    is_deleted,
    email_verified,
    must_change_password
) VALUES (
    gen_random_uuid(),
    'admin@veritas.com',
    crypt('SecurePass123!', gen_salt('bf', 10)),
    'System',
    'Admin',
    'SystemAdmin',
    true,
    false,
    true,
    false
);

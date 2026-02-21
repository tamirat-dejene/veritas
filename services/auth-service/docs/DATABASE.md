# Database Schema

The service uses two primary tables in the `veritas_core` database.

## Table: `veritas_users`

Stores core identity and security audit data for all system administrators and staff.

| Column | Type | Description |
|---|---|---|
| `id` | UUID | Primary Key (PK) |
| `email` | VARCHAR | Unique login identifier |
| `password_hash` | TEXT | bcrypt hashed password |
| `honorific` | VARCHAR | Title (Mr, Ms, etc.) |
| `first_name` | VARCHAR | User's first name |
| `last_name` | VARCHAR | User's last name |
| `role` | VARCHAR | System role (Enum) |
| `enterprise_id` | UUID | Associated tenant/enterprise |
| `is_active` | BOOLEAN | Account enabled status |
| `is_deleted` | BOOLEAN | Soft delete status |
| `failed_login_attempts`| INT | Count of consecutive failed logins |
| `locked_until` | TIMESTAMP | Time until account is unlocked |
| `last_login_at` | TIMESTAMP | Last successful authentication |
| `last_login_ip` | INET | IP address of last login |
| `last_user_agent` | TEXT | Browser agent of last login |
| `created_at` | TIMESTAMP | Record creation date |
| `updated_at` | TIMESTAMP | Last record update date |

---

## Table: `refresh_tokens`

Manages session persistence and revocation.

| Column | Type | Description |
|---|---|---|
| `id` | UUID | Primary Key (PK) |
| `user_id` | UUID | Foreign Key (FK) to `veritas_users` |
| `token_hash` | TEXT | SHA-256 hash of the raw token string |
| `expires_at` | TIMESTAMP | Token expiration date |
| `revoked` | BOOLEAN | Hard revocation flag |
| `created_at` | TIMESTAMP | Creation date |

### Important Implementation Notes

- **One-Way Token Storage**: The service NEVER stores the raw refresh token string. It only stores the `SHA-256` hash. This prevents session hijacking even if the database is compromised.
- **Cascading Deletes**: Refresh tokens are automatically purged if the corresponding user is deleted from the system.

# Auth Service

The `auth-service` is a core microservice of the Veritas multi-tenant SaaS system responsible for user authentication and session management.

## Features

- **Multi-Tenant Support**: Associates users with enterprise IDs.
- **Secure Authentication**: Uses bcrypt for password hashing and HMAC-SHA256 for JWTs.
- **Token Rotation**: Implements secure refresh token rotation and revocation.
- **Account Protection**: Features automatic account locking after consecutive failed login attempts.
- **Audit Logging**: Tracks login events including timestamps, IP addresses, and User Agents.
- **Clean Architecture**: Built with a modular, testable, and maintainable structure.

## Quick Start

### Prerequisites

- Go 1.23+
- PostgreSQL
- Veritas Shared Library

### Running Locally

1. Set up environment variables (see [Configuration](#configuration)).
2. Install dependencies:
   ```bash
   go mod tidy
   ```
3. Run the server:
   ```bash
   go run cmd/server/main.go
   ```

## Configuration

The service is configured via environment variables:

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8081` | HTTP server port |
| `PG_VERITAS_HOST` | `localhost` | PostgreSQL host |
| `PG_VERITAS_PORT` | `5432` | PostgreSQL port |
| `PG_VERITAS_USER` | `postgres` | PostgreSQL user |
| `PG_VERITAS_PASSWORD` | - | PostgreSQL password (Required) |
| `PG_VERITAS_CORE_DB` | `veritas_core` | PostgreSQL database name |
| `PG_SSL_MODE` | `require` | PostgreSQL SSL mode (`require`, `verify-full`, `disable` for local only) |
| `JWT_SECRET` | - | HMAC signing key (Required) |
| `ACCESS_TOKEN_TTL` | `15m` | Access token expiry (e.g., 15m, 1h) |
| `REFRESH_TOKEN_TTL` | `168h` | Refresh token expiry (default 7 days) |

## Documentation Index

- [API Specification](./API.md) - Endpoint details and request/response models.
- [Architecture](./ARCHITECTURE.md) - Design patterns and project structure.
- [Database Schema](./DATABASE.md) - Table definitions and data models.
- [Security Mechanisms](./SECURITY.md) - Details on JWT, hashing, and rotation.

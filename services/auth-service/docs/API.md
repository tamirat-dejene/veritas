# API Specification

The Auth Service exposes REST APIs for authentication management. All requests and responses use JSON.

## Authentication Endpoints

### Error Response Format

All non-2xx responses follow a standard structure:

```json
{
  "code": "invalid_request",
  "message": "invalid request body",
  "requestId": "f47ac10b-58cc-4372-a567-0e02b2c3d479"
}
```

- `code`: Stable, machine-readable error identifier.
- `message`: Human-readable error message.
- `requestId`: Correlation ID (also returned in `X-Request-ID` header).

### Login
Authenticates a user and returns a token pair.

- **URL**: `/auth/login`
- **Method**: `POST`
- **Auth required**: NO

**Request Body**
```json
{
  "email": "user@example.com",
  "password": "secure_password"
}
```

**Success Response (200 OK)**
```json
{
  "accessToken": "...",
  "refreshToken": "...",
  "expiresIn": 900
}
```

**Common Error Responses**
- `401 Unauthorized`: Invalid email or password.
- `403 Forbidden`: Account is locked, inactive, or deleted.
- `400 Bad Request`: Validation errors or malformed JSON.

---

### Refresh Token
Exchanges a valid refresh token for a new access/refresh token pair (Token Rotation).

- **URL**: `/auth/refresh`
- **Method**: `POST`
- **Auth required**: NO (Uses Refresh Token in Body)

**Request Body**
```json
{
  "refreshToken": "b8a54f0f0cc6d2f68dd0b457ea4bb7f814ff69ec487f474f5c6f1781b6f0a0d3"
}
```

**Success Response (200 OK)**
- Same as Login.

**Common Error Responses**
- `401 Unauthorized`: Token is invalid, expired, or revoked.

---

### Logout
Revokes the provided refresh token.

- **URL**: `/auth/logout`
- **Method**: `POST`
- **Auth required**: NO (Uses Refresh Token in Body)

**Request Body**
```json
{
  "refreshToken": "b8a54f0f0cc6d2f68dd0b457ea4bb7f814ff69ec487f474f5c6f1781b6f0a0d3"
}
```

**Success Response (204 No Content)**
- Successful revocation.

**Note**: This endpoint is idempotent. Revoking an already revoked, expired, invalid, or non-existent token returns success (`204`) to prevent information leakage.

---

## Utility Endpoints

### Health Check

- **URL**: `/health`
- **Method**: `GET`

**Response (200 OK)**
```json
{
  "status": "ok"
}
```

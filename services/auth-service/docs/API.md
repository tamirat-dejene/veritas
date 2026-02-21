# API Specification

The Auth Service exposes REST APIs for authentication management. All requests and responses use JSON.

## Authentication Endpoints

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
  "refreshToken": "raw_random_token_string"
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
  "refreshToken": "raw_random_token_string"
}
```

**Success Response (204 No Content)**
- Successful revocation.

**Note**: This endpoint is idempotent. Revoking an already revoked or non-existent token returns success to prevent information leakage.

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

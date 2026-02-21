# Security Mechanisms

The Auth Service implement several layers of defense to protect user credentials and sessions.

## 1. Password Security
- **Hashing**: Uses `bcrypt` for all password storage.
- **Verification**: Constant-time password comparisons to prevent timing attacks.

## 2. JWT Access Tokens
- **Algorithm**: `HMAC-SHA256` (HS256).
- **Scope**: Narrow TTL (default 15m) to limit the window of impact for stolen tokens.
- **Claims**: Includes `sub` (user_id), `email`, `role`, and `enterpriseId`.

## 3. Refresh Token Rotation
The service implements **Refresh Token Rotation**, providing a balance between user convenience (long-lived sessions) and security.

1. On Login, a user receives both an Access Token and a Refresh Token.
2. When the Access Token expires, the client calls `/auth/refresh` with the Refresh Token.
3. The server validates the Refresh Token and **revokes it immediately**.
4. A **brand new** Refresh Token is issued along with a new Access Token.
5. This ensures that if a Refresh Token is leaked, it can only be used once before the legitimate user is affected (at which point both tokens are invalidated).

## 4. Opaque Token Hashing
Opaque refresh tokens are generated using `crypto/rand` (256-bit random). Before storage in the database, these tokens are hashed using `SHA-256`. 
Clients send the raw token, which the server hashes to perform a lookup.

## 5. Account Locking
As a defense against brute-force attacks:
- **Attempts Limit**: 5 consecutive failed login attempts.
- **Lock Duration**: 15 minutes.
- **State Reset**: Successful login resets the counter.

## 6. Information Leakage Prevention
- **Generic Errors**: The service returns a generic "invalid email or password" message for both non-existent users and incorrect passwords to prevent user enumeration.
- **Idempotent Logout**: Logout returns success even if the token hash is not found, preventing attackers from probing valid token hashes.

## 7. Audit Logging
Every authentication event is enriched with metadata:
- Origin IP address.
- Client User Agent.
- Precise timestamps for tracking and suspicious activity analysis.

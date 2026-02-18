# linktag-auth — Production-Ready Go Auth Backend

A production-ready REST API implementing JWT authentication + RBAC with MySQL.

## Tech Stack

| Layer | Choice |
|---|---|
| Language | Go 1.23 |
| HTTP Framework | **Chi** — stdlib-compatible, zero-magic, easy to test |
| Database | MySQL 8.x + **sqlx** |
| Migrations | golang-migrate |
| JWT | golang-jwt/jwt/v5 + **RS256** (asymmetric) |
| Password | **bcrypt** (cost 12) |
| Rate Limiting | golang.org/x/time/rate (per-IP token bucket) |
| Logging | log/slog (stdlib, structured JSON) |

---

## Quick Start (Docker)

### 1. Generate RSA Keys

```bash
make generate-keys
# Copy the base64 output into your .env file
```

### 2. Configure Environment

```bash
cp .env.example .env
# Edit .env — set JWT_PRIVATE_KEY_B64 and JWT_PUBLIC_KEY_B64
```

### 3. Start Services

```bash
docker-compose up --build -d
```

### 4. Verify

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

---

## Run Locally (without Docker)

```bash
# Start MySQL separately, then:
cp .env.example .env  # fill in values
go run ./cmd/api/main.go
```

---

## API Reference

All routes are versioned under `/api/v1`.

### POST /api/v1/auth/register

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"SecurePass123!"}'
```

**Response 201:**
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "status": "active",
    "roles": ["user"],
    "created_at": "2026-02-18T09:00:00Z",
    "updated_at": "2026-02-18T09:00:00Z"
  }
}
```

---

### POST /api/v1/auth/login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"SecurePass123!"}'
```

**Response 200:**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
    "token_type": "Bearer",
    "expires_in": 900,
    "expires_at": "2026-02-18T09:15:00Z"
  }
}
```

---

### POST /api/v1/auth/refresh

```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"eyJ..."}'
```

**Response 200:** Same shape as login response (new token pair).

---

### POST /api/v1/auth/logout

```bash
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"eyJ..."}'
```

**Response 200:**
```json
{"success": true, "data": {"message": "logged out successfully"}}
```

---

### GET /api/v1/auth/me

```bash
curl http://localhost:8080/api/v1/auth/me \
  -H "Authorization: Bearer eyJ..."
```

**Response 200:** Same shape as register response.

---

### GET /api/v1/admin/users *(admin only)*

```bash
curl http://localhost:8080/api/v1/admin/users \
  -H "Authorization: Bearer <admin-access-token>"
```

**Response 200:**
```json
{
  "success": true,
  "data": {
    "users": [...],
    "total": 5
  }
}
```

---

## Error Response Format

All errors follow a consistent envelope:

```json
{
  "success": false,
  "error": {
    "code": "INVALID_CREDENTIALS",
    "message": "invalid email or password"
  }
}
```

| Code | HTTP | Meaning |
|---|---|---|
| `VALIDATION_ERROR` | 422 | Invalid input |
| `EMAIL_TAKEN` | 409 | Duplicate email |
| `INVALID_CREDENTIALS` | 401 | Wrong email/password |
| `ACCOUNT_INACTIVE` | 403 | Account banned/inactive |
| `UNAUTHORIZED` | 401 | Missing/invalid access token |
| `FORBIDDEN` | 403 | Insufficient role |
| `TOKEN_INVALID` | 401 | Refresh token invalid/expired |
| `TOKEN_REVOKED` | 401 | Refresh token revoked |
| `RATE_LIMITED` | 429 | Too many requests |
| `INTERNAL_ERROR` | 500 | Server error |

---

## Running Tests

```bash
# Unit tests (no Docker required)
make test-unit

# Integration tests (requires Docker daemon)
make test-integration

# All tests
make test
```

---

## Database Schema

```
users          — id (UUID), email (unique), password_hash, status, timestamps
roles          — id, name (unique)
user_roles     — user_id FK, role_id FK (many-to-many)
refresh_tokens — jti PK, user_id FK, token_hash (SHA-256), issued_at,
                 expires_at, revoked_at, replaced_by_jti, created_ip, user_agent
```

**RBAC tradeoff:** Many-to-many `user_roles` was chosen over a single `role` column to allow users to hold multiple roles (e.g. `user` + `moderator`). The cost is a JOIN on every role check, which is negligible given the small table size.

---

## JWT Design

```
Access token:  typ=access,  RS256, TTL=15min
               claims: sub, iss, aud, exp, iat, nbf, jti, roles[]

Refresh token: typ=refresh, RS256, TTL=7d
               claims: sub, iss, aud, exp, iat, jti
               (roles intentionally omitted)
```

**Token substitution prevention:** `ValidateAccessToken` rejects `typ=refresh` and `ValidateRefreshToken` rejects `typ=access`. These validators are mutually exclusive.

---

## Threat Model

### Token Theft
- Access tokens are short-lived (15 min). Damage window is limited.
- Refresh tokens are stored as SHA-256 hashes in the DB — raw token is never persisted.
- HTTPS is required in production to prevent interception.

### Replay Attacks
- Each token has a unique `jti`. Refresh tokens are single-use (rotated on every use).
- The old token is revoked in the DB before issuing a new one.
- Token hash is verified against the DB record to prevent forgery.

### Token Rotation
- On every `/refresh` call, the old refresh token is revoked and a new pair is issued.
- `replaced_by_jti` links old → new for audit trail.
- If a revoked token is presented (replay), the server returns 401 immediately.

### Revocation
- Logout revokes the refresh token server-side (sets `revoked_at`).
- Access tokens cannot be revoked before expiry — keep TTL short (15 min).
- For immediate access token revocation, implement a short-lived blocklist (Redis).

### SQL Injection
- All DB queries use parameterized statements (`?` placeholders via sqlx).
- No string concatenation in SQL queries.

### Brute Force
- Login and refresh endpoints are rate-limited (5 req/s per IP, burst 10).
- bcrypt cost 12 makes each hash verification ~250ms, limiting brute-force speed.

### Algorithm Confusion
- JWT parsing enforces `RS256` explicitly — rejects `alg=none` and other algorithms.

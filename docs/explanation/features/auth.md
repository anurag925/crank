---
title: Auth Feature
---

# Auth feature

The `auth` feature adds **JWT-based authentication** with bcrypt password hashing and token revocation. It provides auth endpoints (including logout), a JWT middleware for protecting routes, and value objects for email and password in the domain layer.

## What it provides

| File | Purpose |
|------|---------|
| `internal/domain/user/password.go` | Password value object with bcrypt hashing |
| `internal/domain/user/email.go` | Email value object with validation |
| `internal/ports/hasher.go` | Password hashing interface |
| `internal/ports/tokenservice.go` | Token issue/refresh/validation/revocation interface |
| `internal/ports/tokendenylist.go` | Token revocation denylist port |
| `pkg/crypto/bcrypt_hasher.go` | BCrypt implementation |
| `internal/adapters/auth/jwt/token_service.go` | JWT token management (access + refresh + revocation) |
| `internal/adapters/persistence/gorm/token_denylist.go` | GORM-backed token denylist (when gorm enabled) |
| `internal/adapters/http/web/auth_handler.go` | `/auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout`, `/me` |
| `internal/adapters/http/web/middleware/auth.go` | `JWTAuth()` Echo middleware |

### Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/auth/register` | ❌ | Create account + return tokens |
| POST | `/auth/login` | ❌ | Authenticate + return tokens |
| POST | `/auth/refresh` | ❌ | Exchange refresh token for new pair |
| POST | `/auth/logout` | ✅ Bearer | Revoke refresh token |
| GET | `/me` | ✅ Bearer | Current user identity from JWT |

### Self-scoped user endpoints

User endpoints (`GET/PUT/DELETE /api/v1/users/:id`) verify that the JWT `user_id` matches the path parameter. A mismatch returns 404 (not 403) for IDOR safety. Users can only access their own record.

## Token revocation

When the `gorm` feature is also enabled, a `revoked_tokens` table stores revoked JWT JTIs. The `Refresh` method checks the denylist before issuing new tokens, and `Logout` adds the refresh token's JTI to the denylist.

## Tech stack

| Library | Purpose |
|---------|---------|
| [golang-jwt/jwt v5](https://github.com/golang-jwt/jwt) | JWT signing and validation |
| [golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto) | BCrypt password hashing |
| [google/uuid](https://github.com/google/uuid) | UUID generation for token JTIs |

## Notes

- Protect custom routes with: `e.GET("/admin", handler, middleware.JWTAuth(tokenService))`
- JWT secret should come from `.env`, not `configs/config.yaml`.
- Token expiration settings are configurable under the `jwt` section.
- `pkg/crypto/` is shared with the `crypto` feature (AES-256-GCM cipher).

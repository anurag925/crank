---
title: Auth Feature
---

# Auth feature

The `auth` feature adds **JWT-based authentication** with bcrypt password hashing. It provides auth endpoints, a JWT middleware for protecting routes, and value objects for email and password in the domain layer.

## What it provides

| File | Purpose |
|------|---------|
| `internal/domain/user/password.go` | Password value object with bcrypt hashing |
| `internal/domain/user/email.go` | Email value object with validation |
| `internal/ports/hasher.go` | Password hashing interface |
| `internal/ports/tokenservice.go` | Token issue/refresh/validation interface |
| `internal/adapters/crypto/bcrypt_hasher.go` | BCrypt implementation |
| `internal/adapters/crypto/jwt_token_service.go` | JWT token management (access + refresh) |
| `internal/adapters/http/web/auth_handler.go` | `/auth/register`, `/auth/login`, `/auth/refresh`, `/me` |
| `internal/adapters/http/web/middleware/auth.go` | `JWTAuth()` Echo middleware |

### Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/auth/register` | ❌ | Create account + return tokens |
| POST | `/auth/login` | ❌ | Authenticate + return tokens |
| POST | `/auth/refresh` | ❌ | Exchange refresh token for new pair |
| GET | `/me` | ✅ Bearer | Current user identity from JWT |

## Tech stack

| Library | Purpose | Documentation |
|---------|---------|---------------|
| [golang-jwt/jwt v5](https://github.com/golang-jwt/jwt) | JWT signing and validation | [github.com/golang-jwt/jwt](https://github.com/golang-jwt/jwt) |
| [golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto) | BCrypt password hashing | [pkg.go.dev/golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto) |
| [google/uuid](https://github.com/google/uuid) | UUID generation for token IDs | [github.com/google/uuid](https://github.com/google/uuid) |

## Learning resources

- **JWT.io** — [jwt.io](https://jwt.io) (debugger, algorithm overview, libraries)
- **JWT handbook** — [auth0.com/resources/ebooks/jwt-handbook](https://auth0.com/resources/ebooks/jwt-handbook)
- **BCrypt explained** — [en.wikipedia.org/wiki/Bcrypt](https://en.wikipedia.org/wiki/Bcrypt)
- **Go JWT guide** — [github.com/golang-jwt/jwt](https://github.com/golang-jwt/jwt) (usage examples, signing methods, validation)
- **Best practices for JWT** — [datatracker.ietf.org/doc/html/rfc8725](https://datatracker.ietf.org/doc/html/rfc8725)

## Notes

- Protect custom routes by adding the middleware in Echo: `e.GET("/admin", handler, middleware.JWTAuth(tokenService))`
- JWT secret should come from environment variables, not `configs/config.yaml`.
- Token expiration settings are configurable via `configs/config.yaml` under the `jwt` section.

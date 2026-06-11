# rev — Golang Backend CLI - Specification

## Concept & Vision

A modular CLI tool that scaffolds production-ready Golang backend applications with enterprise-grade infrastructure baked in, and wraps common development tools as subcommands so developers never need to leave the CLI. The bootstrapper generates a well-architected project with clean separation of concerns (models, repositories, services, handlers), sensible defaults, and extensibility through plugin-style feature installation. Think "Laravel Forge meets Go" — a tool that handles the boilerplate so developers focus on business logic.

## Architecture Overview

```
rev/
├── cmd/
│   └── bootstrap/
│       └── main.go           # CLI entry point
├── internal/
│   ├── bootstrap/
│   │   ├── generator.go      # Core code generation logic
│   │   ├── tool.go           # Tool interface + ToolRegistry
│   │   ├── tool_registry.go  # GlobalToolRegistry singleton
│   │   ├── features/         # Feature modules
│   │   │   ├── base/         # Core framework (echo, viper, slog)
│   │   │   ├── auth/         # JWT authentication
│   │   │   ├── postgres/     # Bun ORM + migrations
│   │   │   ├── redis/        # Redis integration (future)
│   │   │   └── mongodb/      # MongoDB integration (future)
│   │   ├── tools/            # Tool wrappers
│   │   │   ├── migrate/      # golang-migrate wrapper
│   │   │   ├── swag/         # swaggo/swag wrapper
│   │   │   ├── build/        # go build wrapper
│   │   │   ├── run/          # go run wrapper
│   │   │   ├── dev/          # air live-reload wrapper
│   │   │   └── ...           # more tool wrappers
│   │   └── template/         # Go template files
│   └── utils/
│       ├── fileutil.go       # File operations
│       └── exec.go           # FindBinary, RunExternal helpers
└── go.mod
```

## Feature List

### Core Features (Built-in)

1. **Base HTTP Server**
   - Echo v4 framework
   - Graceful shutdown
   - Structured logging with slog
   - Health check endpoint (`/health`)
   - Configuration via Viper (configs/config.yaml / env vars)

2. **JWT Authentication**
   - JWT middleware for protected routes
   - Token generation and validation
   - Refresh token support
   - Configurable secret and expiration

3. **Encryption (AES-256-GCM)**
   - Encrypt/decrypt helpers as a reusable library
   - Reads secret from `CRYPTO_SECRET` env var via config
   - Base64-url encoded output, safe for URLs and JSON
   - Unique nonce per encryption call (deterministic output is impossible)

4. **PostgreSQL + Bun ORM**
   - Database connection with Bun
   - Migration support via golang-migrate
   - Repository pattern implementation
   - Transaction support

4. **Project Structure**
   - Clean layered architecture
   - Models with validation
   - Repository interfaces
   - Service layer with dependency injection
   - HTTP handlers
   - Middleware chain

5. **Input Validation**
   - [`go-playground/validator`](https://github.com/go-playground/validator) integration
   - Custom Echo binder that auto-validates on `Bind()`
   - Structured JSON error responses with per-field messages
   - Extensible custom validator registration

6. **Project Structure**
   - Clean layered architecture
   - Models with validation
   - Repository interfaces
   - Service layer with dependency injection
   - HTTP handlers
   - Middleware chain

7. **Development Tools**
   - Air.toml for live reload (debug mode)
   - Makefile with common commands
   - Docker configuration for deployment
   - .env.example template

### Modular Features (Installable)

6. **Redis Integration**
   - Session storage
   - Caching layer
   - Rate limiting support

7. **MongoDB Integration**
   - Document storage
   - Aggregation pipelines

## CLI Commands

```bash
# Initialize new project (tools are checked/installed automatically)
./rev init myapp --features=base,auth,postgres

# Add features to existing project
./rev add redis --project=./myapp
./rev add mongodb --project=./myapp

# List available features
./rev list

# List available tool subcommands
./rev tools

# Generate migration
./rev make migration create_users --project=./myapp

# Run migrations (--project optional; defaults to current directory)
./rev migrate up --project=./myapp
cd myapp && ./rev migrate up

# Run project
./rev run --project=./myapp
cd myapp && ./rev run

# Other tool subcommands (all accept --project or use current directory)
./rev build --project=./myapp
./rev test -v --project=./myapp
./rev swag --project=./myapp
./rev dev --project=./myapp
./rev gofmt --project=./myapp
./rev vet --project=./myapp
./rev tidy --project=./myapp
```

## Generated Project Structure

```
myapp/
├── cmd/
│   └── server/
│       └── main.go           # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go         # Viper configuration
│   ├── handler/
│   │   ├── handler.go        # Base handler with dependencies
│   │   └── user.go           # Example handler
│   ├── middleware/
│   │   ├── auth.go           # JWT middleware
│   │   └── logging.go        # Request logging
│   ├── model/
│   │   └── user.go           # Domain models
│   ├── repository/
│   │   └── user.go           # Data access layer
│   ├── service/
│   │   └── user.go           # Business logic
│   ├── crypto/
│   │   └── crypto.go         # AES-256-GCM Encrypt/Decrypt (if crypto feature enabled)
│   └── validator/
│       ├── validator.go      # Struct validation (go-playground/validator)
│       └── errors.go         # Validation error types & human-readable messages
│   └── database/
│       ├── postgres.go       # DB connection
│       └── migrate.go        # Migration runner
├── configs/                  # Configuration files
│   └── config.yaml           # Application config
├── migrations/
│   ├── 000001_init.up.sql
│   └── 000001_init.down.sql
├── .env.example              # Environment template
├── Makefile                  # Build commands
├── air.toml                  # Live reload config
├── Dockerfile                # Container image
└── go.mod
```

## Configuration Schema (config.yaml)

```yaml
app:
  name: "myapp"
  host: "0.0.0.0"
  port: 8080
  env: "development"

database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "postgres"
  name: "myapp"
  sslmode: "disable"

jwt:
  secret: "your-secret-key-change-in-production"
  expiration: 24h
  refresh_expiration: 168h

logging:
  level: "debug"
  format: "json"
```

## Configuration Strategy

This project uses a dual-file configuration approach with Viper handling priority:

### configs/config.yaml — Application Defaults & Structure
- Database hosts, ports, app name, JWT expiration defaults
- **Safe to commit** to version control
- Documents all available configuration options
- Serves as documentation for developers
- Follows the [`configs/` directory convention](https://github.com/golang-standards/project-layout#configs)

### .env — Local Secrets & Environment Overrides
- Database passwords, JWT secrets, API keys
- **Never committed** (added to .gitignore)
- Loaded by Viper as environment variables

### Viper Priority Order (highest to lowest)
1. **Environment variables** (e.g., `APP_PORT=3000 db_password=secret`)
2. **.env file** (if using dotenv loader)
3. **configs/config.yaml defaults**

This approach allows:
- **Production**: Kubernetes secrets/env vars override configs/config.yaml
- **Local development**: .env file for convenience
- **CI/CD**: Environment variables set by the pipeline

### Bootstrap CLI vs Generated Projects

| File | Bootstrap CLI | Generated Projects |
|------|---------------|-------------------|
| config.yaml | Bootstrapper's own config (install paths, options) | `configs/config.yaml` — full application config with all settings |
| .env | Not needed (CLI doesn't connect to DBs) | `.env` — local secrets (git-ignored) |
| .env.example | Not applicable | `.env.example` — template showing which env vars to set |

## Technical Decisions

| Component | Choice | Rationale |
|-----------|--------|-----------|
| HTTP Framework | Echo v4 | Mature, fast, middleware ecosystem |
| ORM | Bun | Type-safe, fast, migrates well from GORM |
| Config | Viper | ENV vars + YAML support, standard in Go |
| Validation | go-playground/validator | Struct validation with custom rules, human-readable errors |
| API Docs | swaggo/swag + echo-swagger | Swagger/OpenAPI annotations in handler code, UI at /swagger/* |
| Logging | slog | Structured, built-in Go 1.21+ |
| Migrations | golang-migrate | Deterministic, versioned SQL migrations |
| Live Reload | Air | Battle-tested, minimal config |

## Implementation Phases

1. **Phase 1**: Core CLI scaffold + base feature (Echo + Viper + slog)
2. **Phase 2**: Auth feature (JWT middleware + token handling)
3. **Phase 3**: Postgres feature (Bun ORM + migrations)
4. **Phase 4**: Project generation with Makefile + Air + Dockerfile
5. **Phase 5**: Redis module (extensible)
6. **Phase 6**: MongoDB module (extensible)
<div align="center">

<img src="assets/logo.png" alt="crank" width="360">

<p>
  <a href="https://github.com/anurag925/crank/actions/workflows/ci.yml"><img src="https://github.com/anurag925/crank/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/anurag925/crank/releases/latest"><img src="https://img.shields.io/github/v/release/anurag925/crank?color=00ADD8&label=release" alt="Release"></a>
  <a href="https://goreportcard.com/report/github.com/anurag925/crank"><img src="https://goreportcard.com/badge/github.com/anurag925/crank" alt="Go Report Card"></a>
  <img src="https://img.shields.io/github/go-mod/go-version/anurag925/crank?color=00ADD8" alt="Go version">
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License: MIT"></a>
</p>

<p><b>Scaffold production-ready Go backend services — and run your entire dev workflow — without leaving your terminal.</b></p>

</div>

---

`crank` is a modular CLI that scaffolds clean, layered Go backend services from a curated feature set, then wraps everyday tools (`go`, `migrate`, `swag`, `air`, and more) as first-class subcommands.

## Documentation

The full user documentation lives in [`docs/`](docs/index.md) and is built with [docmd](https://docmd.io).

Useful entry points:

- [Getting started](docs/getting-started.md)
- [Commands](docs/commands.md)
- [Features](docs/features.md)
- [Generators](docs/generators.md)
- [Configuration](docs/configuration.md)
- [Troubleshooting](docs/troubleshooting.md)

Run the docs locally:

```bash
npm run docs:dev
```

Build the docs site:

```bash
npm run docs:build
```

## Quick start

```bash
# 1. Install
curl -fsSL https://raw.githubusercontent.com/anurag925/crank/main/install.sh | sh

# 2. Scaffold a service. GORM is the default ORM.
crank init myapp --features=base,auth

# 3. Run it
cd myapp
cp .env.example .env
crank run
```

The generated server listens on `http://localhost:8080` by default.

## Common commands

```bash
crank list                                      # list features
crank tools                                     # list tool wrappers
crank add redis --project ./myapp               # add a feature
crank make scaffold Product title:string --tests --project ./myapp
crank migrate up --project ./myapp
crank test --project ./myapp
crank doctor --project ./myapp
```

## Features

| Feature | Description |
| --- | --- |
| `base` | DDD layout, Echo HTTP server, config, validation, logging, Swagger plumbing, in-memory adapters, Dockerfile, `.air.toml`, and Makefile. |
| `gorm` | PostgreSQL persistence with GORM. Default ORM. |
| `auth` | JWT auth, bcrypt password hashing, auth endpoints, and protected route middleware. |
| `crypto` | AES-256-GCM encryption/decryption helper. |
| `redis` | Redis cache port and go-redis client. |
| `mongodb` | MongoDB client. |
| `qdrant` | Qdrant vector database client. |
| `temporal` | Temporal client, worker, workflows, and activities. |
| `otel` | OpenTelemetry tracing. |
| `outbox` | Transactional outbox for domain events. Requires `gorm`. |
| `views` | React SPA with Vite, embedded by the Go binary. |

## Development

```bash
make build       # build ./bin/crank
make test-unit   # fast unit + integration tests
make test-e2e    # end-to-end tests; needs network
make lint        # gofmt + go vet
```

The helper script offers the same test layers:

```bash
./scripts/test.sh unit
./scripts/test.sh e2e
./scripts/test.sh all
```

## License

Released under the [MIT License](LICENSE) © 2026 Anurag Upadhyay.

---
title: Crypto Feature
---

# Crypto feature

The `crypto` feature adds **AES-256-GCM** authenticated encryption helpers. Use it to encrypt sensitive fields before storage or transport.

## What it provides

| File | Purpose |
|------|---------|
| `internal/ports/cipher.go` | `Encrypt` / `Decrypt` interface |
| `internal/adapters/crypto/aesgcm_cipher.go` | AES-256-GCM implementation using a config-driven secret key |

## Tech stack

| Library | Purpose | Documentation |
|---------|---------|---------------|
| [crypto/aes](https://pkg.go.dev/crypto/aes) | AES encryption (stdlib) | [pkg.go.dev/crypto/aes](https://pkg.go.dev/crypto/aes) |
| [crypto/cipher](https://pkg.go.dev/crypto/cipher) | GCM mode (stdlib) | [pkg.go.dev/crypto/cipher](https://pkg.go.dev/crypto/cipher) |

## Learning resources

- **AES-GCM explained** — [en.wikipedia.org/wiki/Galois/Counter_Mode](https://en.wikipedia.org/wiki/Galois/Counter_Mode)
- **Go crypto package** — [pkg.go.dev/crypto](https://pkg.go.dev/crypto)
- **NIST AES-GCM recommendation** — [csrc.nist.gov](https://csrc.nist.gov/publications/detail/sp/800-38d/final)

## Notes

- Generate a strong 256-bit key: `openssl rand -base64 32`
- The secret key is loaded from configuration (via the `cipher.secret_key` config field).
- No external dependencies are required — it uses Go's standard library.

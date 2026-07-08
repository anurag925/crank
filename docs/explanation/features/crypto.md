---
title: Crypto Feature
---

# Crypto feature

The `crypto` feature adds **AES-256-GCM** authenticated encryption helpers in `pkg/crypto/`. Use it to encrypt sensitive fields before storage or transport.

## What it provides

| File | Purpose |
|------|---------|
| `internal/ports/cipher.go` | `Encrypt` / `Decrypt` interface |
| `pkg/crypto/aesgcm_cipher.go` | AES-256-GCM implementation using a config-driven secret key |

## Notes

- Generate a strong 256-bit key: `openssl rand -base64 32`
- The secret key is loaded from `.env` via `CRYPTO_SECRET`.
- No external dependencies — uses Go's standard library (`crypto/aes`, `crypto/cipher`, `crypto/sha256`).
- `pkg/crypto/` is shared with the `auth` feature's bcrypt hasher.

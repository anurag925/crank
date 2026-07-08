---
title: Qdrant Feature
---

# Qdrant feature

The `qdrant` feature adds a **Qdrant vector database** port and two clients wired into the composition root. Use it for semantic search, embedding storage, and vector similarity retrieval.

## What it provides

| File | Purpose |
|------|---------|
| `internal/ports/qdrant.go` | `Qdrant` port interface (Health, CollectionExists, UpsertPoint, SearchPoints, etc.) |
| `internal/adapters/persistence/qdrant/client.go` | Qdrant gRPC client connection |
| `internal/adapters/persistence/qdrant/http_client.go` | Qdrant REST API client using resty |

## Notes

- Connection settings are in `configs/config.yaml` under the `qdrant` section.
- For local dev: `docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant`.
- Gracefully degrades if unreachable — the server starts without it.

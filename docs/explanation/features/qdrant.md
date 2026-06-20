---
title: Qdrant Feature
---

# Qdrant feature

The `qdrant` feature adds a **Qdrant vector database** gRPC client wired into the composition root. Use it for semantic search, embedding storage, and vector similarity retrieval.

## What it provides

| File | Purpose |
|------|---------|
| `internal/adapters/persistence/qdrant/client.go` | Qdrant gRPC client connection |

## Tech stack

| Library | Purpose | Documentation |
|---------|---------|---------------|
| [qdrant/go-client](https://github.com/qdrant/go-client) | Official Qdrant Go gRPC client | [qdrant.tech/documentation](https://qdrant.tech/documentation/) |
| Qdrant | Vector similarity search engine | [qdrant.tech](https://qdrant.tech) |

## Learning resources

- **Qdrant documentation** — [qdrant.tech/documentation](https://qdrant.tech/documentation/) (quickstart, collections, vectors, filtering)
- **Qdrant Go client** — [github.com/qdrant/qdrant](https://github.com/qdrant/qdrant)
- **Qdrant recipes** — [qdrant.tech/documentation/examples](https://qdrant.tech/documentation/examples/)
- **Vector similarity search guide** — [qdrant.tech/articles](https://qdrant.tech/articles/)

## Notes

- Connection settings are configurable in `configs/config.yaml` under the `qdrant` section.
- Qdrant uses gRPC for communication — the config includes host, port, and API key fields.
- For local development, start Qdrant with Docker: `docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant`.
- Combine with embedding models (e.g., from OpenAI, Cohere, or sentence-transformers) to store and search vectors.

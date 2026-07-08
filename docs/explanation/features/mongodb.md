---
title: MongoDB Feature
---

# MongoDB feature

The `mongodb` feature adds a **MongoDB** client wired into the composition root. Use it for document storage or NoSQL workloads.

## What it provides

| File | Purpose |
|------|---------|
| `internal/adapters/persistence/mongodb/client.go` | Mongo client connection wired as `mdb` |

Access via `mdb.Client().Database("name").Collection("coll")`. Gracefully degrades if MongoDB is unreachable — the server starts without it.

## Notes

- Connection settings in `configs/config.yaml` under the `mongodb` section.
- For local dev: `docker run -p 27017:27017 mongo`.

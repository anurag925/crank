---
title: MongoDB Feature
---

# MongoDB feature

The `mongodb` feature adds a **MongoDB document database** client wired into the composition root. Use it for document storage or workloads that fit a NoSQL data model.

## What it provides

| File | Purpose |
|------|---------|
| `internal/adapters/persistence/mongodb/client.go` | Mongo client connection wired as `mdb` |

Access the database through the wired client:

```go
db := mdb.Client().Database("myapp")
collection := db.Collection("documents")
```

## Tech stack

| Library | Purpose | Documentation |
|---------|---------|---------------|
| [MongoDB Go Driver](https://github.com/mongodb/mongo-go-driver) | Official MongoDB driver | [mongodb.com/docs/drivers/go](https://www.mongodb.com/docs/drivers/go/) |
| MongoDB | Document database | [mongodb.com/docs](https://www.mongodb.com/docs/) |

## Learning resources

- **MongoDB Go Driver docs** — [mongodb.com/docs/drivers/go](https://www.mongodb.com/docs/drivers/go/) (CRUD, aggregation, transactions)
- **MongoDB University (Go)** — [learn.mongodb.com](https://learn.mongodb.com) (free courses)
- **Go Driver GitHub** — [github.com/mongodb/mongo-go-driver](https://github.com/mongodb/mongo-go-driver)
- **MongoDB manual** — [mongodb.com/docs/manual](https://www.mongodb.com/docs/manual/)

## Notes

- Connection settings are configurable in `configs/config.yaml` under the `mongodb` section.
- The client uses the standard `mongo.Connect` with configurable URI, credentials, and timeouts.
- For local development, start MongoDB with Docker: `docker run -p 27017:27017 mongo`.

---
title: Temporal Feature
---

# Temporal feature

The `temporal` feature adds **Temporal workflow orchestration** — client, worker, and activity/workflow adapter layout.

## What it provides

| File | Purpose |
|------|---------|
| `internal/adapters/temporal/client.go` | Temporal client + slog logging bridge |
| `internal/adapters/temporal/worker.go` | Worker with marker-based registration |
| `internal/adapters/temporal/workflow/greeting.go` | Example workflow |
| `internal/adapters/temporal/activity/greeting.go` | Example activity |
| `cmd/worker/main.go` | Standalone worker entry point |

Generate new workflows/activities with `crank make workflow` and `crank make activity`. Gracefully degrades if Temporal is unreachable.

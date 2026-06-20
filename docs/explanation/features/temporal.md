---
title: Temporal Feature
---

# Temporal feature

The `temporal` feature adds **Temporal workflow orchestration** — client, worker, and an activity/workflow adapter layout. Use it for durable background workflows, retries, long-running processes, and distributed orchestration.

## What it provides

| File | Purpose |
|------|---------|
| `internal/adapters/temporal/client.go` | Temporal client + slog logging bridge |
| `internal/adapters/temporal/worker.go` | Worker with marker-based registration for generated workflows/activities |
| `internal/adapters/temporal/workflow/greeting.go` | Example workflow |
| `internal/adapters/temporal/activity/greeting.go` | Example activity |
| `cmd/worker/main.go` | Standalone worker entry point |

### Code generation

Generate new workflows and activities:

```bash
crank add temporal --project ./myapp
crank make workflow OrderFulfillment order_id:uuid
crank make activity ChargeCard amount:float --tests
```

Generated workflows and activities auto-register with the worker via marker comments.

## Tech stack

| Library | Purpose | Documentation |
|---------|---------|---------------|
| [Temporal Go SDK](https://github.com/temporalio/sdk-go) | Temporal client and worker | [docs.temporal.io/dev-guide/go](https://docs.temporal.io/dev-guide/go) |
| Temporal Server | Workflow orchestration engine | [docs.temporal.io](https://docs.temporal.io) |

## Learning resources

- **Temporal Go Dev Guide** — [docs.temporal.io/dev-guide/go](https://docs.temporal.io/dev-guide/go) (getting started, workflows, activities, testing)
- **Temporal samples (Go)** — [github.com/temporalio/samples-go](https://github.com/temporalio/samples-go)
- **Temporal concepts** — [docs.temporal.io/workflows](https://docs.temporal.io/workflows)
- **Temporal Cloud** — [cloud.temporal.io](https://cloud.temporal.io) (managed service)

## Notes

- The Temporal feature requires access to a Temporal Server (local dev: `temporal server start-dev` or Docker).
- The worker is a separate binary (`cmd/worker/main.go`) that registers workflows and activities and polls for tasks.
- Logging is bridged from Temporal SDK to `log/slog` via a custom logger adapter.
- The `crank make workflow` and `crank make activity` generators only work when the `temporal` feature is enabled.

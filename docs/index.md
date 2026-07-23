---
title: Crank — Production-Ready Go Backend CLI
type: concept
---

# Scaffold & Scale Production Go Services

Crank is a modern CLI framework that scaffolds production-ready Go 1.26 backend services using Domain-Driven Design (DDD), CQRS application handlers, and unified development tool wrappers.

[🚀 Get Started](/tutorials/getting-started) &nbsp; [🧠 DDD Architecture](/explanation/architecture)

`Go 1.26` · `Echo v5` · `GORM` · `Temporal`

---

## Quick Start

Get up and running with a single command:

```bash
# 1. Install Crank CLI
curl -fsSL https://raw.githubusercontent.com/anurag925/crank/main/install.sh | sh

# 2. Scaffold a new service with GORM & Auth
crank init bookstore --features=base,auth,gorm

# 3. Start hot-reloading dev server
cd bookstore
cp .env.example .env
crank dev
```

---

## Why Choose Crank?

| Feature | Description |
| --- | --- |
| 🏛️ **Domain-Driven Architecture** | Clean layer separation between aggregate roots, CQRS application handlers, and HTTP adapters. No ORM leakage into business logic. |
| ⚡ **Aggregate Doubles** | Aggregates double as GORM models with raw `uuid.UUID` identifiers. Zero DTO translation boilerplate needed. |
| 🛠️ **Unified Tool Wrappers** | One CLI for `build`, `run`, `test`, `dev` (live reload), `migrate`, `swag`, `gofmt`, `vet`, and `doctor`. |
| 🧱 **Modular Feature System** | Opt-in modules for JWT Auth, Audit Trails, Redis, MongoDB, Qdrant, Temporal, OpenTelemetry, React Views, and Transactional Outbox. |
| ⚙️ **Rails-Style Generators** | Scaffold domain models, repositories, services, versioned Echo HTTP handlers, Temporal workflows, activities, and SQL migrations effortlessly. |
| 🤖 **AI Agent Ready** | Generated projects embed local agent skills (`.agents/skills/`) and strict architectural constraints for autonomous AI coders. |

---

## Core Philosophy

`crank` cleanly separates two concerns:

1. **Project Generation**: `crank init`, `crank add`, and `crank make` generate clean, un-obfuscated Go code directly into your repository.
2. **Project Operation**: `crank run`, `crank test`, `crank migrate`, and `crank make swag` wrap common tools consistently so developers don't have to fiddle with custom shell scripts.

---

## Documentation Roadmap

| Section | Page | Description |
| --- | --- | --- |
| **📚 Tutorials** | [Installation](/tutorials/installation) | Install Crank binary or build from source code. |
| | [Getting Started](/tutorials/getting-started) | Scaffold your first project and inspect the directory structure. |
| | [Building a Full REST Service](/tutorials/building-a-service) | Step-by-step hands-on guide to building an E-Commerce API. |
| | [Temporal Workflows](/tutorials/temporal-workflows) | Async background jobs, activities, and worker registration. |
| **📖 Reference** | [CLI Commands](/reference/commands) | Complete reference for all Crank CLI flags and options. |
| | [Tool Wrappers](/reference/tools) | How Crank wraps `go test`, `air`, `golang-migrate`, `swag`, etc. |
| | [Features Reference](/reference/features) | Available feature modules and dependency requirements. |
| | [Code Generators](/reference/generators) | How `crank make` scaffolds models, repos, handlers, & workflows. |
| **🧠 Explanation** | [DDD Architecture](/explanation/architecture) | Deep-dive into aggregate doubles, `TxRepositories`, and layer boundaries. |
| | [Transactional Outbox](/explanation/outbox-pattern) | Event-driven microservices pattern with guaranteed delivery. |
| **🔧 How-to Guides** | [Testing Guide](/how-to/testing-guide) | Unit testing, handler mocks, integration tests, and E2E suites. |
| | [Migrations & Seeding](/how-to/migrations-and-seeding) | Database migrations and fake data seed generators. |
| | [Troubleshooting](/how-to/troubleshooting) | Common setup questions and diagnostic solutions. |

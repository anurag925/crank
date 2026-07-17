---
title: AI Agent Support
---

# AI agent support

Crank-generated projects include lightweight guidance for AI coding agents so they can recognize the project as Crank-managed and use the right commands.

## Detection

The primary signal is the project manifest:

```text
.crank.yaml
```

If `.crank.yaml` exists at the project root, agents should treat the repository as a Crank project and read the manifest before making workflow assumptions.

The manifest tells tools and agents:

- the project name
- the Go module path
- enabled Crank features
- generator metadata

If `.crank.yaml` is missing, agents should not assume the repository is managed by Crank unless the user explicitly says so.

## Generated agent files

New projects generated with the base feature include:

```text
myapp/
├── .crank.yaml
├── AGENTS.md
└── .agents/
    └── skills/
        └── crank-project/
            └── SKILL.md
```

### `AGENTS.md`

`AGENTS.md` is a short root-level signpost for AI agents. It tells them to:

- use `.crank.yaml` as the source of truth for enabled features
- use the system-installed `crank` command
- avoid assuming a local `./crank` binary exists
- pass `--project` when running commands from outside the project root

### `.agents/skills/crank-project/SKILL.md`

This is a project-local Zed agent skill. Zed agents can load it when working in a Crank-generated project.

The skill explains how to:

- detect a Crank project
- inspect enabled features
- use Crank tool commands
- use `crank make` generators
- add features with `crank add`
- use `crank update-skill` to refresh the skill file
- validate changes with `crank gofmt`, `crank test`, `crank vet`, and `crank doctor`

## Command expectations

Agents should use the system-installed `crank` binary:

```bash
crank run
crank build
crank test
crank gofmt
crank vet
crank tidy
crank swag
crank migrate up
```

They should not use:

```bash
./crank
```

When the agent is already running commands from the generated project root, no `--project` flag is needed.

When running from another directory, use `--project`:

```bash
crank test --project ./myapp
crank run --project ./myapp
crank migrate up --project ./myapp
```

## Feature-aware behavior

Agents should read `.crank.yaml` before assuming optional packages exist.

Examples:

| Feature | What agents can expect |
| --- | --- |
| `gorm` | GORM-backed database support and migrations. |
| `auth` | Authentication routes, middleware, and JWT configuration. |
| `redis` | Redis client/configuration. |
| `mongodb` | MongoDB client/configuration. |
| `qdrant` | Qdrant vector database client/configuration. |
| `temporal` | Temporal workflow/activity support and worker wiring. |
| `crypto` | Encryption helpers and configuration. |

Do not assume every feature is installed. `.crank.yaml` is the source of truth.

## Preservation behavior

Crank generates `AGENTS.md` and `.agents/skills/crank-project/SKILL.md` with skip-if-exists behavior. If a team customizes these files, future generation steps will not overwrite them.

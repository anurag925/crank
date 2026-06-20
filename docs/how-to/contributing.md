---
title: Contributing
---

# Contributing

This page is for contributors working on `crank` itself.

## Local setup

```bash
git clone https://github.com/anurag925/crank.git
cd crank
go mod tidy
make build
./bin/crank --help
```

## Repository layout

```text
crank/
├── cmd/crank/main.go
├── internal/bootstrap/
│   ├── commands/
│   ├── features/
│   ├── scaffold/
│   └── tools/
├── internal/utils/
├── e2e/
├── scripts/
├── docs/
└── go.mod
```

Important areas:

| Area | Purpose |
| --- | --- |
| `cmd/crank/main.go` | Cobra root command, feature/tool blank imports, version metadata. |
| `internal/bootstrap/commands` | CLI subcommands such as `init`, `add`, `make`, `tools`, and Makefile delegation. |
| `internal/bootstrap/features` | Installable feature modules and their embedded templates. |
| `internal/bootstrap/tools` | Tool wrappers for `go`, `migrate`, `swag`, `air`, and in-process checks. |
| `internal/bootstrap/scaffold` | `crank make` generators and templates. |
| `internal/utils` | Filesystem and external process helpers. |
| `e2e` | End-to-end tests behind the `e2e` build tag. |
| `docs` | User-facing documentation built by docmd. |

## Run tests

Fast suite:

```bash
./scripts/test.sh unit
# or
make test-unit
```

End-to-end suite:

```bash
./scripts/test.sh e2e
# or
make test-e2e
```

Everything:

```bash
./scripts/test.sh all
# or
make test-all
```

The e2e suite builds the real binary and compiles generated projects, so it needs network access for generated project dependencies.

## Add a feature

1. Create a package under `internal/bootstrap/features/<name>`.
2. Implement the feature interface.
3. Add templates under `internal/bootstrap/features/<name>/templates`.
4. Register the feature in `init()`.
5. Add a blank import in `cmd/crank/main.go`.
6. Add or update integration/e2e coverage.
7. Update `docs/features.md` and any relevant guides.

Feature interface:

```go
type Feature interface {
    Name() string
    Description() string
    Files() []FileMapping
    Templates() embed.FS
    Dependencies() []string
}
```

Some features may also expose requirements so the generator can enforce feature dependencies.

## Add a tool wrapper

1. Create `internal/bootstrap/tools/<name>/tool.go`.
2. Implement the tool interface.
3. Register it in `init()`.
4. Add a blank import in `cmd/crank/main.go`.
5. Add tests where appropriate.
6. Update `docs/commands.md`.

Tool interface:

```go
type Tool interface {
    Name() string
    Description() string
    LongDescription() string
    BinaryName() string
    InstallCmd() string
    RequiresFeatures() []string
    AddFlags(cmd *cobra.Command)
    Prepare(projectDir string, cmd *cobra.Command) (*ToolInvocation, error)
    Install() error
}
```

The generic command factory handles `--project`, binary lookup, auto-install, and execution.

## Add a generator kind

1. Add a `Kind*` constant in `internal/bootstrap/scaffold/scaffold.go`.
2. Add a plan case in `buildPlan`.
3. Add templates under `internal/bootstrap/scaffold/templates`.
4. Add the kind to the `crank make` command switch.
5. Add tests for generation and wiring.
6. Update `docs/generators.md`.

## Documentation workflow

The docs site uses docmd.

Run locally:

```bash
npm run docs:dev
```

Build locally:

```bash
npm run docs:build
```

The source is `docs/`, config is `docmd.config.json`, and build output is `dist/`.

When you change CLI behavior, update docs in the same pull request.

## Style guidelines

- Keep Go code `gofmt` formatted.
- Wrap errors with useful context: `fmt.Errorf("context: %w", err)`.
- Preserve generated project user edits wherever possible.
- Prefer marker-based injection and idempotent wiring.
- Do not add dependencies unless they are justified by a feature or tool.
- Keep docs examples copy-pasteable.

## Pull request checklist

Before opening a pull request:

```bash
make lint
make test-unit
```

For feature/template/generator changes, also run:

```bash
make test-e2e
```

For docs changes:

```bash
npm run docs:build
```

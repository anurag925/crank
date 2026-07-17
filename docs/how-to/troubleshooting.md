---
title: Troubleshooting
---

# Troubleshooting

This page lists common issues and practical fixes.

## `crank` is not found after installation

Check where the binary was installed and ensure that directory is on your `PATH`.

For Go installs:

```bash
go env GOPATH
```

Add the Go binary directory to your shell profile:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

Then verify:

```bash
crank --version
```

## `crank init` fails during dependency installation

`crank init` runs `go get` and `go mod tidy` in the generated project. Failures are usually caused by:

- no network access
- private module proxy settings
- an unavailable Go proxy
- an old Go version

Try:

```bash
go version
go env GOPROXY
crank tidy --project ./myapp
```

If your environment requires direct module fetching:

```bash
go env -w GOPROXY=direct
```

## `migrate` binary is missing

`crank migrate` attempts to auto-install golang-migrate. If that fails, install it manually:

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

Make sure Go's binary directory is on your `PATH`.

## `crank migrate` cannot determine the database URL

Provide one explicitly:

```bash
crank migrate up --database-url postgres://postgres:postgres@localhost:5432/myapp?sslmode=disable
```

Or set:

```bash
export DATABASE_URL=postgres://postgres:postgres@localhost:5432/myapp?sslmode=disable
crank migrate up
```

Or ensure `configs/config.yaml` contains a valid `database:` section.

## PostgreSQL connection refused

Make sure PostgreSQL is running and the database exists.

Common checks:

```bash
psql postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable
createdb myapp
```

Then run:

```bash
crank migrate up
```

## Swagger docs are missing

Run:

```bash
crank swag
```

Then restart the server and open:

```text
http://localhost:8080/swagger/index.html
```

If the `swag` binary is missing, `crank` should auto-install it. If not, install manually according to the message printed by `crank`.

## `crank make workflow` or `crank make activity` fails

Temporal generators require the `temporal` feature.

Add it first:

```bash
crank add temporal --project ./myapp
```

Then retry:

```bash
crank make workflow OrderFulfillment --project ./myapp
```

## Generated handler is not reachable

Run:

```bash
crank doctor
```

If handler wiring failed, inspect:

```text
internal/adapters/http/web/routes.go
```

Generated handlers are usually registered automatically. If your routes file was heavily edited and markers were removed, `crank` may print a manual wiring hint instead of modifying the file.

## `file already exists`

Generators avoid overwriting primary artifacts unless you pass `--force`.

```bash
crank make handler Product --force
```

Dependency files that already exist are skipped by design.

## `unknown feature`

Run:

```bash
crank list
```

Then use the exact feature name, for example:

```bash
crank add qdrant --project ./myapp
```

## `crank doctor` reports module path mismatch

Compare `.crank.yaml` and `go.mod`:

```text
.crank.yaml     module_path: github.com/acme/myapp
go.mod          module github.com/acme/myapp
```

If you renamed the module, update both files consistently and run:

```bash
crank tidy
crank doctor
```

## docmd build fails for the documentation site

From the `crank` repository root, run:

```bash
npx @docmd/core build
```

If `npx` cannot download packages, check Node/npm installation and network access:

```bash
node --version
npm --version
```

The site source is `docs/`, configured by `docmd.config.json`, and the output directory is `dist/`.

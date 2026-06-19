---
title: Getting Started
---

# Getting started

This guide creates a small Go backend service, runs it locally, and generates a CRUD resource.

## 1. Create a project

Create a service named `bookstore` with the base project and JWT authentication:

```bash
crank init bookstore --features=base,auth
```

By default, `crank` adds the `gorm` feature automatically. The effective feature set is:

```text
base, auth, gorm
```

If you prefer Bun instead of GORM:

```bash
crank init bookstore --features=base,auth --use-bun
```

You can also pick Bun explicitly:

```bash
crank init bookstore --features=base,auth,bun
```

If you pass only `base`, the current CLI still adds the default ORM (`gorm`). Pick Bun explicitly with `--use-bun` when you want Bun instead.

## 2. Enter the project

```bash
cd bookstore
```

The generated project contains a `.crank.yaml` manifest. `crank` uses this file to know the project name, module path, and enabled features.

## 3. Configure local environment

Copy the example environment file:

```bash
cp .env.example .env
```

`configs/config.yaml` contains safe defaults that can be committed. Secrets and machine-specific values belong in `.env` or real environment variables.

If you selected GORM or Bun, make sure PostgreSQL is available and your database values are correct.

Common local values:

```env
DATABASE_URL=postgres://postgres:postgres@localhost:5432/bookstore?sslmode=disable
JWT_SECRET=change-me-in-development
```

## 4. Run the server

```bash
crank run
```

The server starts with:

```text
http://localhost:8080
```

Check the health endpoint:

```bash
curl http://localhost:8080/health
```

## 5. Generate a resource

Generate a complete CRUD resource for books:

```bash
crank make scaffold Book title:string author:string price:float --tests
```

This creates domain, application, persistence, HTTP handler, tests, and route wiring. If the project has GORM or Bun enabled, it also creates a migration unless you pass `--skip-migration`.

Apply migrations:

```bash
crank migrate up
```

Run tests:

```bash
crank test
```

## 6. Generate Swagger docs

```bash
crank swag
```

Then start the server and open:

```text
http://localhost:8080/swagger/index.html
```

## 7. Use live reload during development

```bash
crank dev
```

This uses `air` and the generated `.air.toml` file.

## Common first commands

```bash
crank list                  # show installable features
crank tools                 # show tool wrappers
crank doctor                # check generated project wiring and manifest health
crank test -v               # run tests with verbose output
crank gofmt                 # format generated and hand-written Go code
crank vet                   # run go vet
crank tidy                  # sync go.mod and go.sum
```

## Targeting a project from elsewhere

All tool commands accept `--project`:

```bash
crank run --project ./bookstore
crank test --project ./bookstore
crank migrate up --project ./bookstore
crank make scaffold Author name:string --project ./bookstore
```

When `--project` is omitted, `crank` uses the current directory.

---
title: Recipes
---

# Recipes

Copy these workflows as starting points.

## Create a default API with auth

```bash
crank init api --features=base,auth
cd api
cp .env.example .env
crank run
```

This uses GORM by default.

## Create a Bun-backed API

```bash
crank init api --features=base,auth --use-bun
cd api
cp .env.example .env
crank migrate up
crank run
```

## Add Redis to an existing project

```bash
crank add redis --project ./api
crank tidy --project ./api
crank doctor --project ./api
```

Then update `.env` or your deployment environment with Redis connection values if needed.

## Generate a CRUD resource

```bash
crank make scaffold Product title:string sku:string price:float active:bool --tests --project ./api
crank migrate up --project ./api
crank test --project ./api
```

## Generate only a handler

Use `--only` when you have already written the domain/application layers:

```bash
crank make handler Product --only --project ./api
crank doctor --project ./api
```

## Create a migration manually

```bash
crank make migration add_product_status --project ./api
```

Edit the generated `.up.sql` and `.down.sql` files, then run:

```bash
crank migrate up --project ./api
```

## Add Temporal and generate workflow code

```bash
crank add temporal --project ./api
crank make workflow OrderFulfillment order_id:uuid --project ./api
crank make activity ChargeCard amount:float --tests --project ./api
crank test --project ./api
```

## Add observability

```bash
crank add otel --project ./api
crank run --project ./api
```

The OpenTelemetry feature uses a stdout exporter by default, which is useful locally and easy to replace with a production exporter later.

## Add a React frontend

```bash
crank add views --project ./api
```

The `views` feature adds a React SPA powered by Vite and configures the Go binary to serve embedded frontend assets while preserving development-mode hot reload.

## Generate docs for your API

```bash
crank swag --project ./api
crank run --project ./api
```

Open:

```text
http://localhost:8080/swagger/index.html
```

## Check project health in CI

A simple generated-project CI job can run:

```bash
crank tidy
crank gofmt
crank vet
crank test
crank doctor
```

If you run migrations in CI, provide a database and set:

```bash
DATABASE_URL=postgres://postgres:postgres@localhost:5432/api?sslmode=disable
```

## Build the `crank` docs locally with docmd

The `crank` repository stores documentation in `docs/` and uses docmd to build a static site.

```bash
npm run docs:dev
npm run docs:build
```

The GitHub Pages workflow builds the same site with:

```bash
npx @docmd/core build
```

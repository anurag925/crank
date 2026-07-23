---
title: Building a Full REST Service
type: tutorial
---

# Building a Full REST Service with Crank

In this hands-on tutorial, you will build a complete, production-ready **E-Commerce Product & Order API** from scratch using `crank`. 

You'll see how `crank` handles project setup, entity domain modeling, HTTP routing, database persistence with GORM, automated migrations, seed data generation, and unit testing.

---

## What You'll Build

- **Service Name**: `shopservice`
- **Modules Enabled**: Base Echo API server, JWT Authentication, and PostgreSQL persistence with GORM.
- **Domain Entities**:
  - `Product`: Title (`string`), Price (`float64`), SKU (`string`), Stock (`int`)
  - `Order`: CustomerName (`string`), TotalAmount (`float64`), Status (`string`)

---

## Step 1: Scaffold the Project

Run `crank init` to generate the service foundation with `base`, `auth`, and `gorm` features:

```bash
crank init shopservice --features=base,auth,gorm
cd shopservice
```

This creates a full DDD directory layout with an Echo v5 Web server, Viper configuration, GORM adapter, JWT authentication handlers, and `log/slog` structured logging.

---

## Step 2: Generate the Domain Models & HTTP Endpoints

Instead of manually creating aggregate structs, repository interfaces, GORM implementations, Echo handlers, and route registrations, use `crank make scaffold`:

```bash
# Generate the Product domain aggregate, GORM repository, application handler, and HTTP routes
crank make scaffold Product title:string price:float sku:string stock:int --tests

# Generate the Order domain aggregate, repository, application handler, and HTTP routes
crank make scaffold Order customer_name:string total_amount:float status:string --tests
```

### What Crank Generated:
- **`internal/domain/product/product.go`**: Aggregate root struct using `uuid.UUID` and GORM tags.
- **`internal/adapters/persistence/gorm/product_repository.go`**: GORM aggregate repository with full CRUD operations.
- **`internal/application/uow/unit_of_work.go`**: Updated `TxRepositories` interface including `Products()` and `Orders()`.
- **`internal/adapters/http/web/v1/product_handler.go`**: Versioned Echo HTTP handler at `/api/v1/products`.
- **`migrations/*_create_products.up.sql`**: Auto-generated schema migration.

---

## Step 3: Run Database Migrations

Apply the database migrations to bring your PostgreSQL database up to date:

```bash
# Copy local environment configuration
cp .env.example .env

# Run database migrations
crank migrate up
```

> [!TIP]
> `crank migrate` automatically uses `golang-migrate` behind the scenes, referencing the database connection string from your `.env` file (`DATABASE_URL`).

---

## Step 4: Populate Seed Data

To quickly test your API with realistic data, generate seed records using Crank's built-in seed generator:

```bash
# Generate 25 sample products
crank make seed Product --count 25

# Execute seed insertion into the database
crank make seed up
```

---

## Step 5: Test the REST Endpoints

Start the development server with hot-reloading:

```bash
crank dev
```

Your service is now running on `http://localhost:8080`. Test your newly created endpoints:

### 1. List Products
```bash
curl -X GET http://localhost:8080/api/v1/products
```

**Response (`200 OK`)**:
```json
{
  "data": [
    {
      "id": "9b1deb4d-3b7d-41b3-963f-712c41c7b8d4",
      "title": "Wireless Noise-Canceling Headphones",
      "price": 299.99,
      "sku": "AUDIO-WNC-01",
      "stock": 42,
      "created_at": "2026-07-23T20:00:00Z"
    }
  ]
}
```

### 2. Create a Product
```bash
curl -X POST http://localhost:8080/api/v1/products \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Mechanical Gaming Keyboard",
    "price": 149.50,
    "sku": "KB-MECH-RGB",
    "stock": 15
  }'
```

---

## Step 6: Run Tests

Run the unit test suite across all generated domain layers:

```bash
crank test -v
```

Crank generated unit tests for every handler and repository, ensuring validation logic and CRUD operations remain solid as your code evolves.

---

## Next Steps

- Explore [Temporal Workflows](./temporal-workflows.md) to add asynchronous background processing (e.g. charging cards, sending order emails).
- Read about [Architecture & DDD Patterns](../explanation/architecture.md) to understand Crank's aggregate doubles and `TxRepositories` design.

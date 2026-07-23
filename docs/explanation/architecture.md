---
title: DDD & CQRS Architecture
type: concept
---

# DDD & CQRS Architecture in Crank

`crank` generates Go services structured around **Domain-Driven Design (DDD)** and **Command Query Responsibility Segregation (CQRS)** principles.

Unlike traditional Go boilerplate generators that mix ORM structs with HTTP handlers or leak GORM models into business logic, Crank enforces strict layer boundaries, aggregate doubles, zero ORM leakage, and unit-of-work transaction management.

---

## High-Level Architecture Blueprint

```
                     ┌─────────────────────────────────────────┐
                     │            Composition Root             │
                     │           `cmd/server/main.go`          │
                     └────────────────────┬────────────────────┘
                                          │
                                          ▼
┌───────────────────────────────────────────────────────────────────────────────────┐
│  Adapters Layer                                                                   │
│  ┌───────────────────────────┐    ┌───────────────────────────┐                   │
│  │ HTTP Handlers (Echo v5)   │    │ Persistence Repos (GORM)  │                   │
│  │ `adapters/http/web/v1/`   │    │ `adapters/persistence/`   │                   │
│  └─────────────┬─────────────┘    └─────────────▲─────────────┘                   │
└────────────────┼────────────────────────────────┼─────────────────────────────────┘
                 │ (calls CQRS / UoW)             │ (implements Repositories)
                 ▼                                │
┌─────────────────────────────────────────────────┴─────────────────────────────────┐
│  Application Layer                                                                │
│  ┌─────────────────────────────────────────────────────────────────────────────┐  │
│  │ CQRS Handlers & `TxRepositories` Unit of Work (`application/uow/`)          │  │
│  └──────────────────────────────────────┬──────────────────────────────────────┘  │
└─────────────────────────────────────────┼─────────────────────────────────────────┘
                                          │ (uses pure aggregates)
                                          ▼
┌───────────────────────────────────────────────────────────────────────────────────┐
│  Domain Layer                                                                     │
│  ┌─────────────────────────────────────────────────────────────────────────────┐  │
│  │ Aggregate Roots with `uuid.UUID` & exported domain fields                   │  │
│  └─────────────────────────────────────────────────────────────────────────────┘  │
└───────────────────────────────────────────────────────────────────────────────────┘
```

---

## Core Architectural Pillars

### 1. Aggregate Doubles (Zero DTO Boilerplate)

In traditional DDD implementations, developers often write three parallel struct definitions for every entity:
1. `DomainAggregate`
2. `GormModel` (Row DTO)
3. `HttpRequestDTO`

This leads to endless, error-prone `toAggregate()` and `rowFromAggregate()` conversion functions.

Crank eliminates this boilerplate using the **Aggregate Double Pattern**:
- Aggregate fields are exported (`ID`, `Name`, `Price`, `CreatedAt`).
- GORM tags (`gorm:"primaryKey;type:uuid"`) are declared directly on the domain aggregate struct.
- Raw `uuid.UUID` is used for entity IDs (with `uuid.Nil` validation in constructors).

```go
package user

import (
	"time"
	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name      string    `gorm:"type:varchar(255);not null" json:"name"`
	Email     string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	CreatedAt time.Time `gorm:"not null" json:"created_at"`
}

func NewUser(name, email string) (*User, error) {
	if name == "" {
		return nil, ErrEmptyName
	}
	return &User{
		ID:        uuid.New(),
		Name:      name,
		Email:     email,
		CreatedAt: time.Now().UTC(),
	}, nil
}
```

> [!NOTE]
> The aggregate doubles as the GORM model. No separate Row DTO is generated, saving thousands of lines of translation code while maintaining total type safety.

---

### 2. `TxRepositories` Unit of Work (Zero ORM Leakage)

Application handlers must execute business logic inside transactions without ever importing `gorm.DB` or persistence packages directly.

Crank solves this with the `TxRepositories` interface:

```go
package uow

import (
	"context"
	"myapp/internal/ports"
)

type TxRepositories interface {
	Users() ports.UserRepository
	Orders() ports.OrderRepository
	Products() ports.ProductRepository
}

type UnitOfWork interface {
	Do(ctx context.Context, fn func(repos TxRepositories) error) error
}
```

#### Application Handler Usage:
```go
func (h *OrderHandler) CreateOrder(ctx context.Context, cmd CreateOrderCommand) error {
	return h.uow.Do(ctx, func(repos uow.TxRepositories) error {
		// 1. Fetch user & product using repository interfaces
		user, err := repos.Users().FindByID(ctx, cmd.UserID)
		if err != nil {
			return err
		}

		product, err := repos.Products().FindByID(ctx, cmd.ProductID)
		if err != nil {
			return err
		}

		// 2. Create order aggregate
		order, err := domain.NewOrder(user.ID, product.ID, product.Price)
		if err != nil {
			return err
		}

		// 3. Save inside the same atomic database transaction
		return repos.Orders().Save(ctx, order)
	})
}
```

---

### 3. Self-Scoped Echo v5 HTTP Adapters

HTTP handlers live in `internal/adapters/http/web/v1/` and are versioned under `/api/v1`.

Each handler:
- Binds incoming requests using `echo.Context.Bind()`.
- Validates request payloads automatically using `go-playground/validator`.
- Returns structured JSON responses wrapped in the standard `api.Response` / `api.Error` envelope.
- Self-registers its own Echo routes via an exported `Register(g *echo.Group)` method.

```go
func (h *UserHandler) Register(g *echo.Group) {
	users := g.Group("/users")
	users.GET("", h.List)
	users.POST("", h.Create)
	users.GET("/:id", h.GetByID)
}
```

---

## Layer Isolation Rules

| Layer | Can Import | CANNOT Import |
| --- | --- | --- |
| **Domain** | Standard library (`uuid`, `time`) | Application, Adapters, Frameworks, GORM, Echo |
| **Application** | Domain, Ports, UoW interfaces | GORM, Echo, Web Frameworks, Database Drivers |
| **Adapters** | Domain, Application, Ports | Other specific adapter implementations |
| **Cmd (Composition Root)** | All layers | — |

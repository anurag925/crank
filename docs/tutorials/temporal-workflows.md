---
title: Temporal Workflows & Workers
type: tutorial
---

# Temporal Workflows & Workers in Crank

`crank` includes native support for [Temporal](https://temporal.io), allowing you to build resilient, distributed background workflows, saga patterns, and async task processing with zero boilerplate.

---

## Overview

When you initialize a project with `--features=temporal`, Crank generates:
- A dedicated Temporal client instance integrated into your composition root (`cmd/server/main.go`).
- A background worker runner in `internal/adapters/worker/worker.go`.
- Subcommands to quickly generate new **Workflows** and **Activities**.

```
myapp/
├── internal/
│   ├── application/
│   │   └── workflow/               # Temporal Workflows
│   │       └── order_fulfillment.go
│   └── adapters/
│       ├── activity/               # Temporal Activities
│       │   └── charge_card.go
│       └── worker/                 # Worker Registration
│           └── worker.go
```

---

## Step 1: Add Temporal to an Existing Project

If you haven't enabled `temporal` during `crank init`, add it using `crank add`:

```bash
crank add temporal
```

This installs the Temporal Go SDK (`go.temporal.io/sdk`) and writes the worker layout.

---

## Step 2: Generate a Workflow and Activity

Use `crank make` generators to generate your business workflow and activities:

```bash
# Generate a ChargeCard Activity
crank make activity ChargeCard amount:float card_token:string --tests

# Generate an OrderFulfillment Workflow that invokes the activity
crank make workflow OrderFulfillment order_id:uuid customer_id:uuid --tests
```

### Auto-Wiring Magic:
Crank automatically registers `ChargeCardActivity` and `OrderFulfillmentWorkflow` in `internal/adapters/worker/worker.go` so the background worker is ready to handle executions immediately.

---

## Step 3: Inspecting Generated Code

### Activity (`internal/adapters/activity/charge_card.go`)
```go
package activity

import (
	"context"
	"fmt"
	"go.uber.org/zap"
)

type ChargeCardInput struct {
	Amount    float64 `json:"amount"`
	CardToken string  `json:"card_token"`
}

type ChargeCardOutput struct {
	TransactionID string `json:"transaction_id"`
	Success       bool   `json:"success"`
}

type ChargeCardActivity struct{}

func NewChargeCardActivity() *ChargeCardActivity {
	return &ChargeCardActivity{}
}

func (a *ChargeCardActivity) Execute(ctx context.Context, input ChargeCardInput) (ChargeCardOutput, error) {
	// Execute payment gateway call
	txnID := fmt.Sprintf("txn_%s", input.CardToken)
	return ChargeCardOutput{TransactionID: txnID, Success: true}, nil
}
```

### Workflow (`internal/application/workflow/order_fulfillment.go`)
```go
package workflow

import (
	"time"
	"go.temporal.io/sdk/workflow"
	"myapp/internal/adapters/activity"
)

type OrderFulfillmentInput struct {
	OrderID    string  `json:"order_id"`
	CustomerID string  `json:"customer_id"`
	Amount     float64 `json:"amount"`
}

func OrderFulfillmentWorkflow(ctx workflow.Context, input OrderFulfillmentInput) error {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var chargeResult activity.ChargeCardOutput
	var act *activity.ChargeCardActivity

	err := workflow.ExecuteActivity(ctx, act.Execute, activity.ChargeCardInput{
		Amount:    input.Amount,
		CardToken: "tok_visa_sample",
	}).Get(ctx, &chargeResult)

	if err != nil {
		return err
	}

	return nil
}
```

---

## Step 4: Triggering Workflows from Echo Handlers

You can inject the Temporal Client into your Echo application handlers to trigger async workflows from HTTP requests:

```go
func (h *OrderHandler) CreateOrder(c echo.Context) error {
	var req CreateOrderRequest
	if err := c.Bind(&req); err != nil {
		return api.Error(c, http.StatusBadRequest, "Invalid request payload")
	}

	workflowOptions := client.StartWorkflowOptions{
		ID:        "order_" + req.OrderID,
		TaskQueue: "shopservice-queue",
	}

	we, err := h.temporalClient.ExecuteWorkflow(
		c.Request().Context(), 
		workflowOptions, 
		workflow.OrderFulfillmentWorkflow, 
		workflow.OrderFulfillmentInput{
			OrderID: req.OrderID,
			Amount:  req.TotalAmount,
		},
	)
	if err != nil {
		return api.Error(c, http.StatusInternalServerError, "Failed to start fulfillment workflow")
	}

	return c.JSON(http.StatusAccepted, map[string]string{
		"workflow_id": we.GetID(),
		"run_id":      we.GetRunID(),
	})
}
```

---

## Step 5: Running the Temporal Worker

Start your service with Temporal enabled:

```bash
# Ensure local Temporal dev server is running (e.g., temporal server start-dev)
crank dev
```

Crank's composition root starts both the Echo HTTP server and the Temporal Worker concurrently, allowing background workflows to process tasks seamlessly.

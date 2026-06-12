APP     := crank
MODULE  := github.com/anurag925/crank
BIN     := ./bin/$(APP)
GO      := go
GOFLAGS :=

# Packages for fast (network-free) tests.
UNIT_PKGS := ./internal/... ./cmd/...

.PHONY: help build clean install test test-unit test-e2e test-cover fmt vet lint tidy snapshot run

## ── Build ────────────────────────────────────────────────────────────────────

help: ## Show this help
	@printf "\nUsage: make [target]\n\n"
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo

build: tidy ## Build the crank binary
	@mkdir -p bin
	$(GO) build $(GOFLAGS) -o $(BIN) ./cmd/$(APP)

clean: ## Remove build artifacts
	rm -rf bin/ dist/ coverage.out

install: build ## Build and install to GOBIN (default ~/go/bin)
	$(GO) install $(GOFLAGS) ./cmd/$(APP)

## ── Test ─────────────────────────────────────────────────────────────────────

test: test-unit ## Alias for test-unit

test-unit: ## Run unit + integration tests (fast, no network)
	$(GO) test $(GOFLAGS) -count=1 $(UNIT_PKGS)

test-e2e: ## Run end-to-end tests (slow, needs network)
	$(GO) test -tags e2e -count=1 -timeout 20m ./e2e/...

test-all: test-unit test-e2e ## Run all test suites

test-cover: ## Run tests with coverage report → coverage.out
	$(GO) test $(GOFLAGS) -coverprofile=coverage.out $(UNIT_PKGS)
	$(GO) tool cover -func=coverage.out | tail -n 1
	@echo "HTML report: go tool cover -html=coverage.out"

## ── Quality ──────────────────────────────────────────────────────────────────

fmt: ## Format all Go source files
	gofmt -s -w .

vet: ## Run go vet
	$(GO) vet $(UNIT_PKGS)

lint: fmt vet ## Format + vet

tidy: ## Run go mod tidy
	$(GO) mod tidy

## ── Release ─────────────────────────────────────────────────────────────────

snapshot: ## Build a local snapshot release via goreleaser
	goreleaser release --snapshot --clean

## ── Run ─────────────────────────────────────────────────────────────────────

run: build ## Build and show help
	$(BIN) --help

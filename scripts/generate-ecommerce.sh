#!/usr/bin/env bash
# generate-ecommerce.sh — end-to-end demo generator.
#
# Builds the crank CLI, scaffolds a full e-commerce backend that exercises
# EVERY feature and EVERY `crank make` kind, seeds fake data, then validates
# the generated project (tidy → gofmt → build → vet → test).
#
# The script performs ZERO manual edits — everything is produced by crank so
# the output can be reviewed as-is.
#
# Requirements: network access (crank init / go get / go mod tidy pull deps).
#
# Usage:
#   ./scripts/generate-ecommerce.sh
#
# Output:
#   tmp/ecommerce/            the generated project (git-ignored)
#   tmp/crank                 the freshly built CLI binary (git-ignored)

set -euo pipefail

# ---------------------------------------------------------------------------
# Setup
# ---------------------------------------------------------------------------
cd "$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
ROOT="$(pwd)"

PROJECT_NAME="ecommerce"
TARGET_DIR="tmp"
PROJECT_DIR="$TARGET_DIR/$PROJECT_NAME"
BIN="$ROOT/tmp/crank"
MODULE="github.com/example/ecommerce"

# All installable features. base is implicit; gorm is required by
# outbox/audit/views. Every feature is mutually compatible.
FEATURES="base,auth,crypto,gorm,redis,mongodb,qdrant,temporal,otel,outbox,audit,views"

# ANSI helpers (fall back to plain text when stdout is not a TTY).
if [ -t 1 ]; then
    BOLD=$(printf '\033[1m'); GREEN=$(printf '\033[32m')
    RED=$(printf '\033[31m'); CYAN=$(printf '\033[36m'); RESET=$(printf '\033[0m')
else
    BOLD=""; GREEN=""; RED=""; CYAN=""; RESET=""
fi

step() { printf '\n%s==> %s%s\n' "$BOLD$CYAN" "$*" "$RESET"; }
ok()   { printf '%s  ✔ %s%s\n' "$GREEN" "$*" "$RESET"; }
die()  { printf '%s  ✗ %s%s\n' "$RED" "$*" "$RESET" >&2; exit 1; }

# crank <args...> — run the built CLI.
crank() { "$BIN" "$@"; }

# mk <kind> <name> [fields...] — thin wrapper around `crank make ... --project`.
mk() {
    local kind="$1"; shift
    printf '%s  • make %-9s %s%s\n' "$BOLD" "$kind" "$*" "$RESET"
    crank make "$kind" "$@" --project "$PROJECT_DIR"
}

# ---------------------------------------------------------------------------
# 1. Build the CLI
# ---------------------------------------------------------------------------
step "Building crank CLI → $BIN"
mkdir -p "$ROOT/tmp"
go build -o "$BIN" ./cmd/crank
ok "built $($BIN --version 2>/dev/null | head -n1 || echo crank)"

# ---------------------------------------------------------------------------
# 2. Fresh project
# ---------------------------------------------------------------------------
step "Removing any previous generation at $PROJECT_DIR"
rm -rf "$PROJECT_DIR"
ok "clean"

step "crank init ($FEATURES)"
crank init "$PROJECT_NAME" \
    --target "$TARGET_DIR" \
    --module "$MODULE" \
    --features="$FEATURES" \
    --force
ok "project scaffolded at $PROJECT_DIR"

# ---------------------------------------------------------------------------
# 3. Scaffold the e-commerce domain — exercise every `make` kind
# ---------------------------------------------------------------------------
step "Scaffolding domain (full stack + tests)"
mk scaffold Product   name:string description:text price:float sku:string stock:int --tests
mk scaffold Category  name:string slug:string --tests
mk scaffold Customer  name:string email:email --tests
mk scaffold Order     customer_id:uuid total:float status:string --tests
mk scaffold Cart      customer_id:uuid --tests
mk scaffold Review    product_id:uuid rating:int comment:text --tests

step "Individual layers (model / repository / service / handler)"
# model: a standalone domain aggregate (+ create-table migration).
mk model      OrderItem order_id:uuid product_id:uuid quantity:int price:float
# repository: a standalone persistence layer (domain + gorm/in-memory adapters).
mk repository Inventory  product_id:uuid quantity:int
# repository + service on the SAME resource: the repository is generated first
# so the GORM adapter (gorm.NewPaymentRepository) exists before `service` wires
# it into the Unit of Work. Running `service` on its own would reference a
# repository that hasn't been generated yet.
mk repository Payment    order_id:uuid amount:float provider:string
mk service    Payment
# handler (full stack): a brand-new resource, all layers generated + route wired.
mk handler    Wishlist   customer_id:uuid product_id:uuid
# handler --only: regenerate JUST the HTTP adapter for an already-scaffolded
# resource (Product), so its domain/application deps already exist.
mk handler    Product --only --force

step "Temporal workflow + activity (requires temporal feature)"
mk workflow OrderFulfillment order_id:uuid
mk activity ChargeCard       amount:float --tests

step "Standalone migration"
mk migration add_index_to_orders

# ---------------------------------------------------------------------------
# 4. Seed data
# ---------------------------------------------------------------------------
step "Generating seed scaffolding + fake data"
crank make seed --project "$PROJECT_DIR"
crank make seed Product  --count 20 --project "$PROJECT_DIR"
crank make seed Category --count 8  --project "$PROJECT_DIR"
crank make seed Customer --count 15 --project "$PROJECT_DIR"
ok "seed files generated"

# ---------------------------------------------------------------------------
# 5. Validate the generated project
# ---------------------------------------------------------------------------
# Each step is reported independently; a failure aborts (set -e) after the
# summary line so the offending command is obvious.
run_step() {
    local label="$1"; shift
    step "$label"
    if "$@"; then
        ok "$label passed"
    else
        die "$label FAILED — command: $*"
    fi
}

run_step "crank tidy (resolve deps)" crank tidy --project "$PROJECT_DIR"
run_step "crank gofmt"               crank gofmt --project "$PROJECT_DIR"
run_step "crank build"               crank build --project "$PROJECT_DIR"
run_step "crank vet"                 crank vet   --project "$PROJECT_DIR"
run_step "crank test"                crank test  --project "$PROJECT_DIR"

# ---------------------------------------------------------------------------
# 6. Summary
# ---------------------------------------------------------------------------
step "Summary"
FILE_COUNT=$(find "$PROJECT_DIR" -type f ! -path '*/.git/*' | wc -l | tr -d ' ')
GO_COUNT=$(find "$PROJECT_DIR" -name '*.go' | wc -l | tr -d ' ')
MIG_COUNT=$(find "$PROJECT_DIR/db/migrations" -name '*.sql' 2>/dev/null | wc -l | tr -d ' ')

printf '%s\n' "  Project:     $PROJECT_DIR"
printf '%s\n' "  Module:      $MODULE"
printf '%s\n' "  Features:    $FEATURES"
printf '%s\n' "  Total files: $FILE_COUNT   Go files: $GO_COUNT   Migrations: $MIG_COUNT"
printf '\n'
printf '%s\n' "  Generated HTTP handlers (internal/adapters/http/web/v1):"
find "$PROJECT_DIR/internal/adapters/http/web/v1" -name '*.go' 2>/dev/null \
    | sed "s#^$PROJECT_DIR/#    #" | sort || true
printf '\n'
ok "e-commerce project generated & validated successfully"
printf '\n%sReview it with:%s  cd %s\n' "$BOLD" "$RESET" "$PROJECT_DIR"

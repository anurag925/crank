package scaffold_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/anurag925/crank/internal/bootstrap"
	"github.com/anurag925/crank/internal/bootstrap/scaffold"

	// Register features so bootstrap.Generate can build a project.
	_ "github.com/anurag925/crank/internal/bootstrap/features/base"
	_ "github.com/anurag925/crank/internal/bootstrap/features/bun"
	_ "github.com/anurag925/crank/internal/bootstrap/features/gorm"
	_ "github.com/anurag925/crank/internal/bootstrap/features/temporal"
)

// newProject scaffolds a fresh crank project into a temp dir and returns its path.
func newProject(t *testing.T, features []string) string {
	t.Helper()
	tmp := t.TempDir()
	res, err := bootstrap.Generate(bootstrap.GlobalRegistry, bootstrap.Options{
		ProjectName: "demo",
		ModulePath:  "github.com/example/demo",
		TargetDir:   tmp,
		Features:    features,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return res.ProjectDir
}

func read(t *testing.T, dir, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}

func exists(dir, rel string) bool {
	_, err := os.Stat(filepath.Join(dir, rel))
	return err == nil
}

// assertParses checks that a generated Go file is syntactically valid.
func assertParses(t *testing.T, dir, rel string) {
	t.Helper()
	path := filepath.Join(dir, rel)
	if _, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.AllErrors); err != nil {
		t.Errorf("generated %s does not parse: %v", rel, err)
	}
}

// dddHandlerLayers is the set of files the KindHandler / KindScaffold
// generators produce. They live under the project's DDD-shaped directory
// tree (domain, application, adapters/persistence, adapters/http/web).
func dddHandlerLayers() []string {
	return []string{
		"internal/domain/order/order.go",
		"internal/domain/order/events.go",
		"internal/domain/order/errors.go",
		"internal/domain/order/repository.go",
		"internal/application/order/commands.go",
		"internal/application/order/command_handler.go",
		"internal/application/order/queries.go",
		"internal/application/order/query_handler.go",
		"internal/adapters/persistence/memory/order_repository.go",
		"internal/adapters/persistence/bun/order_repository.go",
		"internal/adapters/http/web/v1/order_handler.go",
	}
}

func TestGenerateHandlerPostgres(t *testing.T) {
	dir := newProject(t, []string{"bun"})

	res, err := scaffold.Generate(scaffold.Options{
		ProjectDir: dir,
		Kind:       scaffold.KindHandler,
		Name:       "Order",
		Fields:     []string{"customer:string", "total:float"},
	})
	if err != nil {
		t.Fatalf("Generate handler: %v", err)
	}

	for _, rel := range dddHandlerLayers() {
		if !exists(dir, rel) {
			t.Errorf("expected %s to be generated", rel)
			continue
		}
		assertParses(t, dir, rel)
	}

	// Domain aggregate is plain Go — no JSON/DB/validation tags.
	aggregate := read(t, dir, "internal/domain/order/order.go")
	for _, banned := range []string{`json:"`, `bun:"`, `validate:"`} {
		if strings.Contains(aggregate, banned) {
			t.Errorf("domain aggregate must not carry %q tags:\n%s", banned, aggregate)
		}
	}
	if !strings.Contains(aggregate, "func NewOrder(") {
		t.Errorf("aggregate missing NewOrder constructor")
	}

	// Postgres adapter has its own row DTO and maps sql.ErrNoRows to the
	// domain sentinel.
	repo := read(t, dir, "internal/adapters/persistence/bun/order_repository.go")
	for _, want := range []string{
		"type orderRow struct",
		"func NewOrderRepository(db bun.IDB)",
		"func (r *OrderRepository) Save(",
		"func (r *OrderRepository) Delete(",
		"ErrOrderNotFound",
	} {
		if !strings.Contains(repo, want) {
			t.Errorf("bun repository missing %q", want)
		}
	}

	// HTTP handler depends only on the application layer and exposes a
	// Register method.
	handler := read(t, dir, "internal/adapters/http/web/v1/order_handler.go")
	for _, want := range []string{
		"func NewOrderHandler(",
		"func (h *OrderHandler) Register(g *echo.Group)",
		"order.ErrOrderNotFound",
		"application/order",
	} {
		if !strings.Contains(handler, want) {
			t.Errorf("handler missing %q", want)
		}
	}

	// A create-table migration was produced.
	ups, _ := filepath.Glob(filepath.Join(dir, "migrations", "*_create_orders.up.sql"))
	if len(ups) != 1 {
		t.Fatalf("expected one create_orders.up.sql migration, found %d", len(ups))
	}
	up, _ := os.ReadFile(ups[0])
	if !strings.Contains(string(up), "CREATE TABLE IF NOT EXISTS orders") {
		t.Errorf("migration missing CREATE TABLE:\n%s", up)
	}
	if !strings.Contains(string(up), "customer TEXT NOT NULL") {
		t.Errorf("migration missing customer column:\n%s", up)
	}

	// The handler was wired into the central routes aggregator.
	if !res.Wired {
		t.Errorf("expected handler to be wired, hint=%q", res.WireHint)
	}
	hub := read(t, dir, "internal/adapters/http/web/v1/routes.go")
	for _, want := range []string{"*OrderHandler", `e.Group("/orders")`, "cfg.OrderHandler.Register("} {
		if !strings.Contains(hub, want) {
			t.Errorf("routes.go not wired with %q:\n%s", want, hub)
		}
	}
	assertParses(t, dir, "internal/adapters/http/web/v1/routes.go")
}

func TestGenerateHandlerGorm(t *testing.T) {
	dir := newProject(t, []string{"gorm"})

	res, err := scaffold.Generate(scaffold.Options{
		ProjectDir: dir,
		Kind:       scaffold.KindHandler,
		Name:       "Invoice",
		Fields:     []string{"number:string", "amount:float"},
	})
	if err != nil {
		t.Fatalf("Generate handler on gorm project: %v", err)
	}

	// GORM adapter exists.
	gormRepo := "internal/adapters/persistence/gorm/invoice_repository.go"
	if !exists(dir, gormRepo) {
		t.Errorf("expected %s to be generated", gormRepo)
	} else {
		assertParses(t, dir, gormRepo)
	}

	// Bun adapter should NOT exist.
	if exists(dir, "internal/adapters/persistence/bun/invoice_repository.go") {
		t.Error("bun adapter should not be generated on a gorm project")
	}

	// In-memory adapter always ships.
	assertParses(t, dir, "internal/adapters/persistence/memory/invoice_repository.go")

	// Repository uses gorm.DB and GORM tags.
	repo := read(t, dir, gormRepo)
	for _, want := range []string{
		"*gorm.DB",
		"func NewInvoiceRepository(db *gorm.DB)",
		"func (r *InvoiceRepository) Save(",
		"func (r *InvoiceRepository) Delete(",
		"gorm.ErrRecordNotFound",
		"TableName",
	} {
		if !strings.Contains(repo, want) {
			t.Errorf("gorm repository missing %q", want)
		}
	}

	// A create-table migration was produced.
	ups, _ := filepath.Glob(filepath.Join(dir, "migrations", "*_create_invoices.up.sql"))
	if len(ups) != 1 {
		t.Fatalf("expected one create_invoices.up.sql migration, found %d", len(ups))
	}
	up, _ := os.ReadFile(ups[0])
	if !strings.Contains(string(up), "CREATE TABLE IF NOT EXISTS invoices") {
		t.Errorf("migration missing CREATE TABLE: %s", up)
	}

	// The handler was wired.
	if !res.Wired {
		t.Errorf("expected handler to be wired, hint=%q", res.WireHint)
	}
	hub := read(t, dir, "internal/adapters/http/web/v1/routes.go")
	for _, want := range []string{"*InvoiceHandler", `e.Group("/invoices")`, "cfg.InvoiceHandler.Register("} {
		if !strings.Contains(hub, want) {
			t.Errorf("routes.go not wired with %q: %s", want, hub)
		}
	}
	assertParses(t, dir, "internal/adapters/http/web/v1/routes.go")
}

func TestGenerateHandlerInMemory(t *testing.T) {
	dir := newProject(t, nil) // base only, no bun

	res, err := scaffold.Generate(scaffold.Options{
		ProjectDir: dir,
		Kind:       scaffold.KindHandler,
		Name:       "Ticket",
	})
	if err != nil {
		t.Fatalf("Generate handler: %v", err)
	}

	// All DDD layers are generated, but the persistence adapter is the
	// in-memory one (no bun adapter is produced without the feature).
	for _, rel := range []string{
		"internal/domain/ticket/ticket.go",
		"internal/application/ticket/command_handler.go",
		"internal/adapters/persistence/memory/ticket_repository.go",
		"internal/adapters/http/web/v1/ticket_handler.go",
	} {
		if !exists(dir, rel) {
			t.Errorf("expected %s", rel)
		}
	}
	if exists(dir, "internal/adapters/persistence/bun/ticket_repository.go") {
		t.Error("did not expect a bun adapter without the bun feature")
	}
	// No migration without bun.
	ups, _ := filepath.Glob(filepath.Join(dir, "migrations", "*_create_tickets.up.sql"))
	if len(ups) != 0 {
		t.Errorf("did not expect a migration for a non-bun project, found %d", len(ups))
	}

	// In-memory repository is produced regardless of bun.
	assertParses(t, dir, "internal/adapters/persistence/memory/ticket_repository.go")

	if !res.Wired {
		t.Errorf("expected wiring, hint=%q", res.WireHint)
	}
}

func TestGenerateModelOnly(t *testing.T) {
	dir := newProject(t, []string{"bun"})

	_, err := scaffold.Generate(scaffold.Options{
		ProjectDir: dir,
		Kind:       scaffold.KindModel,
		Name:       "Tag",
	})
	if err != nil {
		t.Fatalf("Generate model: %v", err)
	}

	// Model kind emits the domain layer (aggregate, events, errors, port)
	// and nothing else.
	if !exists(dir, "internal/domain/tag/tag.go") {
		t.Error("expected domain/tag/tag.go")
	}
	if exists(dir, "internal/adapters/persistence/memory/tag_repository.go") ||
		exists(dir, "internal/adapters/http/web/v1tag_handler.go") {
		t.Error("model generation should not produce adapters")
	}
}

func TestHandlerOnlySkipsDependencies(t *testing.T) {
	dir := newProject(t, []string{"bun"})

	_, err := scaffold.Generate(scaffold.Options{
		ProjectDir: dir,
		Kind:       scaffold.KindHandler,
		Name:       "Coupon",
		Only:       true,
	})
	if err != nil {
		t.Fatalf("Generate handler --only: %v", err)
	}

	if !exists(dir, "internal/adapters/http/web/v1/coupon_handler.go") {
		t.Error("expected adapter/http/web/v1/coupon_handler.go")
	}
	if exists(dir, "internal/domain/coupon/coupon.go") ||
		exists(dir, "internal/adapters/persistence/memory/coupon_repository.go") {
		t.Error("--only should not generate domain or persistence")
	}
}

func TestPrimaryConflictRequiresForce(t *testing.T) {
	dir := newProject(t, []string{"bun"})
	opts := scaffold.Options{ProjectDir: dir, Kind: scaffold.KindModel, Name: "Note"}

	if _, err := scaffold.Generate(opts); err != nil {
		t.Fatalf("first generate: %v", err)
	}
	if _, err := scaffold.Generate(opts); err == nil {
		t.Fatal("expected an error regenerating an existing model without --force")
	}

	opts.Force = true
	if _, err := scaffold.Generate(opts); err != nil {
		t.Fatalf("regenerate with --force: %v", err)
	}
}

func TestWiringIsIdempotent(t *testing.T) {
	dir := newProject(t, []string{"bun"})
	base := scaffold.Options{ProjectDir: dir, Kind: scaffold.KindHandler, Name: "Review"}

	if _, err := scaffold.Generate(base); err != nil {
		t.Fatalf("first generate: %v", err)
	}
	// Regenerate the handler with --force; wiring must not be duplicated.
	forced := base
	forced.Force = true
	if _, err := scaffold.Generate(forced); err != nil {
		t.Fatalf("regenerate: %v", err)
	}

	hub := read(t, dir, "internal/adapters/http/web/v1/routes.go")
	if n := strings.Count(hub, "cfg.ReviewHandler.Register("); n != 1 {
		t.Errorf("expected exactly one review registration, got %d:\n%s", n, hub)
	}
}

func TestGenerateWithTests(t *testing.T) {
	for _, tc := range []struct {
		name     string
		features []string
	}{
		{"bun", []string{"bun"}},
		{"in_memory", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := newProject(t, tc.features)

			res, err := scaffold.Generate(scaffold.Options{
				ProjectDir: dir,
				Kind:       scaffold.KindScaffold,
				Name:       "Order",
				Fields:     []string{"customer:string", "total:float", "ship_at:time"},
				Tests:      true,
			})
			if err != nil {
				t.Fatalf("Generate with tests: %v", err)
			}

			// Every generated layer gets a syntactically valid test file.
			want := []string{
				"internal/domain/order/order_test.go",
				"internal/domain/order/events_test.go",
				"internal/application/order/commands_test.go",
				"internal/application/order/command_handler_test.go",
				"internal/application/order/queries_test.go",
				"internal/application/order/query_handler_test.go",
				"internal/adapters/persistence/memory/order_repository_test.go",
				"internal/adapters/http/web/v1/order_handler_test.go",
			}
			if tc.features != nil {
				want = append(want, "internal/adapters/persistence/bun/order_repository_test.go")
			}
			for _, rel := range want {
				if !exists(dir, rel) {
					t.Errorf("expected test file %s", rel)
					continue
				}
				assertParses(t, dir, rel)
				if !containsAny(res.Created, rel) {
					t.Errorf("%s not reported as created", rel)
				}
			}

			// The time field must be exercised by the in-memory handler test.
			if tc.features == nil {
				handlerTest := read(t, dir, "internal/adapters/http/web/v1/order_handler_test.go")
				if !strings.Contains(handlerTest, "time.Now()") {
					t.Errorf("expected time.Now() sample in handler test:\n%s", handlerTest)
				}
			}
		})
	}
}

func TestGenerateWithoutTestsOmitsTestFiles(t *testing.T) {
	dir := newProject(t, []string{"bun"})

	if _, err := scaffold.Generate(scaffold.Options{
		ProjectDir: dir,
		Kind:       scaffold.KindModel,
		Name:       "Tag",
	}); err != nil {
		t.Fatalf("Generate model: %v", err)
	}
	if exists(dir, "internal/domain/tag/tag_test.go") {
		t.Error("did not expect a test file without --tests")
	}
}

func containsAny(list []string, target string) bool {
	return slices.Contains(list, target)
}

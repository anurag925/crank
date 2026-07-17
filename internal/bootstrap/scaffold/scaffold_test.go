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
		"internal/adapters/persistence/gorm/order_repository.go",
		"internal/adapters/http/web/v1/order_handler.go",
	}
}

func TestGenerateHandlerGorm(t *testing.T) {
	dir := newProject(t, []string{"gorm"})

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

	// Domain aggregate has exported fields with GORM tags.
	aggregate := read(t, dir, "internal/domain/order/order.go")
	for _, want := range []string{
		"type Order struct",
		"ID        uuid.UUID",
		"Customer  string",
		"Total     float64",
		"CreatedAt time.Time",
		"UpdatedAt time.Time",
		`gorm:"column:id;primaryKey;type:uuid"`,
		`gorm:"column:customer;not null;type:TEXT"`,
		`gorm:"column:total;not null;type:DOUBLE PRECISION"`,
		"func NewOrder(",
		"func (o *Order) TableName() string",
	} {
		if !strings.Contains(aggregate, want) {
			t.Errorf("aggregate missing %q", want)
		}
	}
	// id, created_at and updated_at are grouped together at the top of the
	// struct, ahead of the resource-specific fields.
	if idIdx, cIdx, uIdx, custIdx := strings.Index(aggregate, "ID        uuid.UUID"),
		strings.Index(aggregate, "CreatedAt time.Time"),
		strings.Index(aggregate, "UpdatedAt time.Time"),
		strings.Index(aggregate, "Customer  string"); !(idIdx >= 0 && idIdx < cIdx && cIdx < uIdx && uIdx < custIdx) {
		t.Errorf("expected id/created_at/updated_at grouped at the top of the struct:\n%s", aggregate)
	}
	// No getters allowed.
	for _, banned := range []string{"func (o *Order) ID()", "func (o *Order) Customer()", "func (o *Order) Total()"} {
		if strings.Contains(aggregate, banned) {
			t.Errorf("aggregate must not have getter %q:\n%s", banned, aggregate)
		}
	}

	// GORM adapter uses the aggregate directly (no Row DTO).
	repo := read(t, dir, "internal/adapters/persistence/gorm/order_repository.go")
	for _, want := range []string{
		"func NewOrderRepository(db *gorm.DB)",
		"func (r *OrderRepository) Save(",
		"func (r *OrderRepository) Delete(",
		"gorm.ErrRecordNotFound",
	} {
		if !strings.Contains(repo, want) {
			t.Errorf("gorm repository missing %q", want)
		}
	}
	for _, banned := range []string{"orderRow", "toAggregate", "OrderRowFromAggregate"} {
		if strings.Contains(repo, banned) {
			t.Errorf("gorm repository must not contain %q", banned)
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
	ups, _ := filepath.Glob(filepath.Join(dir, "db/migrations", "*_create_orders.up.sql"))
	if len(ups) != 1 {
		t.Fatalf("expected one create_orders.up.sql migration, found %d", len(ups))
	}
	up, _ := os.ReadFile(ups[0])
	if !strings.Contains(string(up), "CREATE TABLE IF NOT EXISTS orders") {
		t.Errorf("migration missing CREATE TABLE:\n%s", up)
	}
	if !strings.Contains(string(up),
		"id UUID PRIMARY KEY,\n    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,\n    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,") {
		t.Errorf("migration must group id/created_at/updated_at at the top:\n%s", up)
	}
	if !strings.Contains(string(up), "customer TEXT NOT NULL") {
		t.Errorf("migration missing customer column:\n%s", up)
	}

	// The handler was wired into the central routes aggregator.
	if !res.Wired {
		t.Errorf("expected handler to be wired, hint=%q", res.WireHint)
	}
	hub := read(t, dir, "internal/adapters/http/web/v1/routes.go")
	for _, want := range []string{"*OrderHandler", `g.Group("/orders")`, "cfg.OrderHandler.Register("} {
		if !strings.Contains(hub, want) {
			t.Errorf("routes.go not wired with %q:\n%s", want, hub)
		}
	}
	assertParses(t, dir, "internal/adapters/http/web/v1/routes.go")
}

func TestGenerateHandlerGormDetail(t *testing.T) {
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
	} {
		if !strings.Contains(repo, want) {
			t.Errorf("gorm repository missing %q", want)
		}
	}

	// A create-table migration was produced.
	ups, _ := filepath.Glob(filepath.Join(dir, "db/migrations", "*_create_invoices.up.sql"))
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
	for _, want := range []string{"*InvoiceHandler", `g.Group("/invoices")`, "cfg.InvoiceHandler.Register("} {
		if !strings.Contains(hub, want) {
			t.Errorf("routes.go not wired with %q: %s", want, hub)
		}
	}
	assertParses(t, dir, "internal/adapters/http/web/v1/routes.go")
}

func TestGenerateHandlerInMemory(t *testing.T) {
	dir := newProject(t, nil) // base only, no gorm

	res, err := scaffold.Generate(scaffold.Options{
		ProjectDir: dir,
		Kind:       scaffold.KindHandler,
		Name:       "Ticket",
	})
	if err != nil {
		t.Fatalf("Generate handler: %v", err)
	}

	// All DDD layers are generated, but the persistence adapter is the
	// in-memory one (no ORM adapter is produced without the feature).
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
	if exists(dir, "internal/adapters/persistence/gorm/ticket_repository.go") {
		t.Error("did not expect a gorm adapter without the gorm feature")
	}
	// No migration without gorm.
	ups, _ := filepath.Glob(filepath.Join(dir, "db/migrations", "*_create_tickets.up.sql"))
	if len(ups) != 0 {
		t.Errorf("did not expect a migration for a non-ORM project, found %d", len(ups))
	}

	// In-memory repository is produced regardless of ORM.
	assertParses(t, dir, "internal/adapters/persistence/memory/ticket_repository.go")

	if !res.Wired {
		t.Errorf("expected wiring, hint=%q", res.WireHint)
	}
}

// TestHandlerWiresCompositionRoot guards the composition-root wiring: a
// generated handler must be constructed AND passed to v1.Mount in
// cmd/server/main.go, otherwise its MountConfig field stays nil and every
// route it registers nil-panics at runtime. The in-memory path additionally
// exercises the functional-option wiring into the in-memory UnitOfWork.
func TestHandlerWiresCompositionRoot(t *testing.T) {
	cases := []struct {
		name     string
		features []string
		repoLine string
		wantOpt  bool // in-memory UoW option only present without outbox
	}{
		{"in_memory", nil, "ticketRepo := memory.NewTicketRepository()", true},
		{"gorm", []string{"gorm"}, "ticketRepo := gorm.NewTicketRepository(gormDB)", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := newProject(t, tc.features)
			if _, err := scaffold.Generate(scaffold.Options{
				ProjectDir: dir,
				Kind:       scaffold.KindHandler,
				Name:       "Ticket",
			}); err != nil {
				t.Fatalf("Generate handler: %v", err)
			}

			main := read(t, dir, "cmd/server/main.go")
			want := []string{
				`ticketapp "github.com/example/demo/internal/application/ticket"`,
				tc.repoLine,
				"ticketCmd := ticketapp.NewCommandHandler(ticketRepo, uow)",
				"ticketQry := ticketapp.NewQueryHandler(ticketRepo)",
				"ticketHandler := v1.NewTicketHandler(ticketCmd, ticketQry)",
				"TicketHandler: ticketHandler,",
			}
			if tc.wantOpt {
				want = append(want, "uow.WithTicketRepo(ticketRepo),")
			}
			for _, w := range want {
				if !strings.Contains(main, w) {
					t.Errorf("main.go missing composition wiring %q:\n%s", w, main)
				}
			}
			assertParses(t, dir, "cmd/server/main.go")
		})
	}
}

func TestGenerateModelOnly(t *testing.T) {
	dir := newProject(t, []string{"gorm"})

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
		exists(dir, "internal/adapters/http/web/v1/tag_handler.go") {
		t.Error("model generation should not produce adapters")
	}
}

func TestHandlerOnlySkipsDependencies(t *testing.T) {
	dir := newProject(t, []string{"gorm"})

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
	dir := newProject(t, []string{"gorm"})
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
	dir := newProject(t, []string{"gorm"})
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
		{"gorm", []string{"gorm"}},
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
				want = append(want, "internal/adapters/persistence/gorm/order_repository_test.go")
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
	dir := newProject(t, []string{"gorm"})

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

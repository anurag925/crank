package scaffold_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anurag925/crank/internal/bootstrap"
	"github.com/anurag925/crank/internal/bootstrap/scaffold"

	// Register features so bootstrap.Generate can build a project.
	_ "github.com/anurag925/crank/internal/bootstrap/features/base"
	_ "github.com/anurag925/crank/internal/bootstrap/features/postgres"
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

func TestGenerateHandlerPostgres(t *testing.T) {
	dir := newProject(t, []string{"postgres"})

	res, err := scaffold.Generate(scaffold.Options{
		ProjectDir: dir,
		Kind:       scaffold.KindHandler,
		Name:       "Order",
		Fields:     []string{"customer:string", "total:float"},
	})
	if err != nil {
		t.Fatalf("Generate handler: %v", err)
	}

	for _, rel := range []string{
		"internal/model/order.go",
		"internal/repository/order.go",
		"internal/service/order.go",
		"internal/handler/order.go",
	} {
		if !exists(dir, rel) {
			t.Errorf("expected %s to be generated", rel)
		}
		assertParses(t, dir, rel)
	}

	// Model carries Bun tags and the supplied fields.
	model := read(t, dir, "internal/model/order.go")
	for _, want := range []string{"bun.BaseModel", `bun:"table:orders`, `bun:"customer,notnull"`, `bun:"total,notnull"`} {
		if !strings.Contains(model, want) {
			t.Errorf("model missing %q:\n%s", want, model)
		}
	}

	// Repository is Bun-backed with full CRUD.
	repo := read(t, dir, "internal/repository/order.go")
	for _, want := range []string{"func NewOrderRepository(db *bun.DB)", "func (o *OrderRepository) Update", "func (o *OrderRepository) Delete"} {
		if !strings.Contains(repo, want) {
			t.Errorf("repository missing %q", want)
		}
	}

	// Handler exposes the REST routes.
	handler := read(t, dir, "internal/handler/order.go")
	for _, want := range []string{`e.Group("/orders")`, "repository.NewOrderRepository(deps.DB)", "repository.ErrOrderNotFound"} {
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

	// The handler was wired into the central aggregator.
	if !res.Wired {
		t.Errorf("expected handler to be wired, hint=%q", res.WireHint)
	}
	hub := read(t, dir, "internal/handler/handler.go")
	for _, want := range []string{"orders *OrderHandler", "orders: NewOrderHandler(deps)", "h.orders.Register(e)"} {
		if !strings.Contains(hub, want) {
			t.Errorf("handler.go not wired with %q:\n%s", want, hub)
		}
	}
	assertParses(t, dir, "internal/handler/handler.go")
}

func TestGenerateHandlerInMemory(t *testing.T) {
	dir := newProject(t, nil) // base only, no postgres

	res, err := scaffold.Generate(scaffold.Options{
		ProjectDir: dir,
		Kind:       scaffold.KindHandler,
		Name:       "Ticket",
	})
	if err != nil {
		t.Fatalf("Generate handler: %v", err)
	}

	// Both repository and service are generated regardless of postgres.
	if !exists(dir, "internal/service/ticket.go") {
		t.Error("expected service/ticket.go")
	}
	if !exists(dir, "internal/repository/ticket.go") {
		t.Error("expected repository/ticket.go")
	}
	// No migration without postgres.
	ups, _ := filepath.Glob(filepath.Join(dir, "migrations", "*_create_tickets.up.sql"))
	if len(ups) != 0 {
		t.Errorf("did not expect a migration for a non-postgres project, found %d", len(ups))
	}

	handler := read(t, dir, "internal/handler/ticket.go")
	if !strings.Contains(handler, "service.NewTicketService()") {
		t.Errorf("handler should use the service constructor:\n%s", handler)
	}
	assertParses(t, dir, "internal/handler/ticket.go")
	assertParses(t, dir, "internal/service/ticket.go")

	if !res.Wired {
		t.Errorf("expected wiring, hint=%q", res.WireHint)
	}
}

func TestGenerateModelOnly(t *testing.T) {
	dir := newProject(t, []string{"postgres"})

	_, err := scaffold.Generate(scaffold.Options{
		ProjectDir: dir,
		Kind:       scaffold.KindModel,
		Name:       "Tag",
	})
	if err != nil {
		t.Fatalf("Generate model: %v", err)
	}

	if !exists(dir, "internal/model/tag.go") {
		t.Error("expected model/tag.go")
	}
	if exists(dir, "internal/repository/tag.go") || exists(dir, "internal/handler/tag.go") {
		t.Error("model generation should not produce repository or handler")
	}
}

func TestHandlerOnlySkipsDependencies(t *testing.T) {
	dir := newProject(t, []string{"postgres"})

	_, err := scaffold.Generate(scaffold.Options{
		ProjectDir: dir,
		Kind:       scaffold.KindHandler,
		Name:       "Coupon",
		Only:       true,
	})
	if err != nil {
		t.Fatalf("Generate handler --only: %v", err)
	}

	if !exists(dir, "internal/handler/coupon.go") {
		t.Error("expected handler/coupon.go")
	}
	if exists(dir, "internal/model/coupon.go") || exists(dir, "internal/repository/coupon.go") || exists(dir, "internal/service/coupon.go") {
		t.Error("--only should not generate model, repository or service")
	}
}

func TestPrimaryConflictRequiresForce(t *testing.T) {
	dir := newProject(t, []string{"postgres"})
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
	dir := newProject(t, []string{"postgres"})
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

	hub := read(t, dir, "internal/handler/handler.go")
	if n := strings.Count(hub, "h.reviews.Register(e)"); n != 1 {
		t.Errorf("expected exactly one review registration, got %d:\n%s", n, hub)
	}
}

func TestGenerateWithTests(t *testing.T) {
	for _, tc := range []struct {
		name     string
		features []string
	}{
		{"postgres", []string{"postgres"}},
		{"in_memory", nil},
	} {
		tc := tc
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
				"internal/model/order_test.go",
				"internal/repository/order_test.go",
				"internal/service/order_test.go",
				"internal/handler/order_test.go",
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

			// The time field must be imported and used by the sample helper in the
			// in-memory handler test (postgres handler tests are route-only).
			if tc.features == nil {
				handlerTest := read(t, dir, "internal/handler/order_test.go")
				if !strings.Contains(handlerTest, "time.Now()") {
					t.Errorf("expected time.Now() sample in handler test:\n%s", handlerTest)
				}
			}
		})
	}
}

func TestGenerateWithoutTestsOmitsTestFiles(t *testing.T) {
	dir := newProject(t, []string{"postgres"})

	if _, err := scaffold.Generate(scaffold.Options{
		ProjectDir: dir,
		Kind:       scaffold.KindModel,
		Name:       "Tag",
	}); err != nil {
		t.Fatalf("Generate model: %v", err)
	}
	if exists(dir, "internal/model/tag_test.go") {
		t.Error("did not expect a test file without --tests")
	}
}

func containsAny(list []string, target string) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}

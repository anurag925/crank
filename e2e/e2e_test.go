package e2e

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/anurag925/crank/internal/bootstrap"

	// Register all features with the global registry via init().
	_ "github.com/anurag925/crank/internal/bootstrap/features/auth"
	_ "github.com/anurag925/crank/internal/bootstrap/features/base"
	_ "github.com/anurag925/crank/internal/bootstrap/features/crypto"
	_ "github.com/anurag925/crank/internal/bootstrap/features/mongodb"
	_ "github.com/anurag925/crank/internal/bootstrap/features/postgres"
	_ "github.com/anurag925/crank/internal/bootstrap/features/redis"
	_ "github.com/anurag925/crank/internal/bootstrap/features/temporal"
)

// crankBin is the path to the crank binary built once in TestMain.
var crankBin string

// allFeatureNames lists every feature the application ships, used to validate
// the `crank list` output and to drive the "all features" compile test.
var allFeatureNames = []string{"base", "auth", "crypto", "postgres", "redis", "mongodb", "temporal"}

// allToolNames lists every tool subcommand the application wraps.
var allToolNames = []string{"migrate", "swag", "build", "run", "dev", "test", "gofmt", "vet", "tidy"}

func TestMain(m *testing.M) {
	root := moduleRoot()

	binDir, err := os.MkdirTemp("", "crank-e2e-bin")
	if err != nil {
		panic("create temp bin dir: " + err.Error())
	}
	crankBin = filepath.Join(binDir, "crank")

	build := exec.Command("go", "build", "-o", crankBin, "./cmd/crank")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		os.RemoveAll(binDir)
		panic("build crank binary failed: " + err.Error() + "\n" + string(out))
	}

	code := m.Run()
	os.RemoveAll(binDir)
	os.Exit(code)
}

// moduleRoot returns the repository root (parent of this e2e directory).
func moduleRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("cannot determine caller for module root")
	}
	return filepath.Dir(filepath.Dir(file))
}

// runCrank runs the compiled crank binary with the given args and returns its
// combined output. It fails the test if the command errors.
func runCrank(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := runCrankRaw(t, dir, args...)
	if err != nil {
		t.Fatalf("crank %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return out
}

func runCrankRaw(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(crankBin, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// runGo runs a go subcommand inside dir and fails the test on error.
func runGo(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go %s failed in %s: %v\n%s", strings.Join(args, " "), dir, err, out)
	}
}

// ==========================================================================
// CLI surface — exercises the real binary end to end (no side effects).
// ==========================================================================

func TestE2E_Version(t *testing.T) {
	out := runCrank(t, "", "--version")
	if !strings.Contains(out, "crank version") {
		t.Errorf("--version output missing 'crank version': %q", out)
	}
}

func TestE2E_Help(t *testing.T) {
	out := runCrank(t, "", "--help")
	for _, want := range []string{"init", "add", "list", "make", "tools"} {
		if !strings.Contains(out, want) {
			t.Errorf("--help output missing command %q\n%s", want, out)
		}
	}
}

func TestE2E_List_ShowsAllFeatures(t *testing.T) {
	out := runCrank(t, "", "list")
	for _, f := range allFeatureNames {
		if !strings.Contains(out, f) {
			t.Errorf("`crank list` output missing feature %q\n%s", f, out)
		}
	}
}

func TestE2E_List_JSON(t *testing.T) {
	out := runCrank(t, "", "list", "--json")
	var entries []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal([]byte(out), &entries); err != nil {
		t.Fatalf("`crank list --json` produced invalid JSON: %v\n%s", err, out)
	}
	got := make(map[string]bool, len(entries))
	for _, e := range entries {
		if e.Description == "" {
			t.Errorf("feature %q has empty description in JSON output", e.Name)
		}
		got[e.Name] = true
	}
	for _, f := range allFeatureNames {
		if !got[f] {
			t.Errorf("`crank list --json` missing feature %q", f)
		}
	}
}

func TestE2E_Tools_ShowsAllTools(t *testing.T) {
	out := runCrank(t, "", "tools")
	for _, tool := range allToolNames {
		if !strings.Contains(out, tool) {
			t.Errorf("`crank tools` output missing tool %q\n%s", tool, out)
		}
	}
}

func TestE2E_UnknownCommand_Fails(t *testing.T) {
	out, err := runCrankRaw(t, "", "definitely-not-a-command")
	if err == nil {
		t.Errorf("expected unknown command to fail, got success:\n%s", out)
	}
}

// ==========================================================================
// Generation + compilation — proves generated projects build and vet cleanly.
// ==========================================================================

// compileCases is the curated matrix of feature combinations whose generated
// output is fully compiled. It deliberately covers each feature at least once
// plus a heavy multi-feature combination.
var compileCases = []struct {
	name     string
	features []string
}{
	{"base_only", []string{"base"}},
	{"auth", []string{"auth"}},
	{"postgres", []string{"postgres"}},
	{"redis", []string{"redis"}},
	{"mongodb", []string{"mongodb"}},
	{"crypto", []string{"crypto"}},
	{"temporal", []string{"temporal"}},
	{"auth_postgres_crypto", []string{"auth", "postgres", "crypto"}},
	{"all", allFeatureNames},
}

func TestE2E_GenerateAndCompile(t *testing.T) {
	for _, tc := range compileCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			projectDir := scaffold(t, "svc_"+tc.name, tc.features)
			compileProject(t, projectDir)
		})
	}
}

// TestE2E_Make exercises the full `crank make` generator surface through the
// real binary on top of a generated project and proves that everything it emits
// — models, repositories/services, handlers, router wiring, migrations and the
// companion _test.go files — compiles, vets and passes its own tests.
//
// It runs against both data-layer variants:
//   - postgres: Bun-backed repositories + create-table migrations.
//   - base only: in-memory services and no migrations.
func TestE2E_Make(t *testing.T) {
	cases := []struct {
		name     string
		features []string
		postgres bool
	}{
		{"postgres", []string{"postgres"}, true},
		{"base_only", []string{"base"}, false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			projectDir := scaffold(t, "make_"+tc.name, tc.features)

			// 1. Full stack with mixed field types + generated tests.
			runCrank(t, "", "make", "scaffold", "Order", "customer:string", "total:float", "paid:bool", "--tests", "--project", projectDir)
			// 2. Every supported field type in a single resource, with tests, to
			//    prove the Go/SQL type mapping and validation samples all hold up.
			runCrank(t, "", "make", "scaffold", "Account",
				"name:string", "bio:text", "age:int", "balance:int64", "rating:float",
				"active:bool", "joined_at:time", "token:uuid", "contact:email",
				"--tests", "--project", projectDir)
			// 3. Multi-word resource via handler (pulls in model + repo/service + wiring).
			runCrank(t, "", "make", "handler", "OrderItem", "label:string", "--project", projectDir)
			// 4. Standalone single-layer generators (each pulls in its model).
			runCrank(t, "", "make", "model", "Category", "name:string", "--project", projectDir)
			runCrank(t, "", "make", "repository", "Ticket", "subject:string", "--project", projectDir)
			runCrank(t, "", "make", "service", "Cart", "label:string", "--project", projectDir)
			// 5. A standalone, DB-agnostic migration pair.
			runCrank(t, "", "make", "migration", "add_index_to_orders", "--project", projectDir)

			// Every resource has a model (note the multi-word snake_case file name).
			for _, rel := range []string{
				"internal/model/order.go",
				"internal/model/account.go",
				"internal/model/order_item.go",
				"internal/model/category.go",
				"internal/model/ticket.go",
				"internal/model/cart.go",
			} {
				assertExists(t, projectDir, rel)
			}

			// Handlers exist and are wired into the central aggregator.
			for _, rel := range []string{
				"internal/handler/order.go",
				"internal/handler/account.go",
				"internal/handler/order_item.go",
			} {
				assertExists(t, projectDir, rel)
			}
			hub := readFile(t, projectDir, "internal/handler/handler.go")
			for _, want := range []string{
				"h.orders.Register(e)",
				"h.accounts.Register(e)",
				"h.orderItems.Register(e)",
			} {
				if !strings.Contains(hub, want) {
					t.Errorf("handler.go missing registration %q", want)
				}
			}

			// --tests produced a companion test for every layer of the two
			// scaffolds (and only those — the bare generators omit them).
			for _, rel := range []string{
				"internal/model/order_test.go",
				"internal/handler/order_test.go",
				"internal/model/account_test.go",
				"internal/handler/account_test.go",
			} {
				assertExists(t, projectDir, rel)
			}
			assertNotExists(t, projectDir, "internal/handler/order_item_test.go")
			assertNotExists(t, projectDir, "internal/model/category_test.go")

			// Data-layer placement and migration behavior depend on the DB feature.
			if tc.postgres {
				assertExists(t, projectDir, "internal/repository/order.go")
				assertExists(t, projectDir, "internal/repository/account.go")
				assertExists(t, projectDir, "internal/repository/ticket.go")
				assertNotExists(t, projectDir, "internal/service/order.go")
				// Postgres resources get create-table migrations...
				for _, name := range []string{"orders", "accounts", "order_items", "categories", "tickets"} {
					if n := globCount(t, projectDir, "migrations/*_create_"+name+".up.sql"); n != 1 {
						t.Errorf("expected exactly one create_%s migration, found %d", name, n)
					}
				}
				// ...but an in-memory `service` never produces one.
				if n := globCount(t, projectDir, "migrations/*_create_carts.up.sql"); n != 0 {
					t.Errorf("did not expect a carts migration, found %d", n)
				}
			} else {
				assertExists(t, projectDir, "internal/service/order.go")
				assertExists(t, projectDir, "internal/service/account.go")
				assertExists(t, projectDir, "internal/service/cart.go")
				assertNotExists(t, projectDir, "internal/repository/order.go")
				// Non-postgres scaffolds never emit create-table migrations.
				if n := globCount(t, projectDir, "migrations/*_create_*.up.sql"); n != 0 {
					t.Errorf("did not expect create-table migrations for a non-postgres project, found %d", n)
				}
			}

			// The standalone migration is DB-agnostic and always created as a pair.
			if n := globCount(t, projectDir, "migrations/*_add_index_to_orders.up.sql"); n != 1 {
				t.Errorf("expected one add_index_to_orders.up.sql migration, found %d", n)
			}
			if n := globCount(t, projectDir, "migrations/*_add_index_to_orders.down.sql"); n != 1 {
				t.Errorf("expected one add_index_to_orders.down.sql migration, found %d", n)
			}

			// The whole project — including every generated _test.go — must compile,
			// vet and pass.
			compileProject(t, projectDir)
			runGo(t, projectDir, "test", "./internal/...")
		})
	}
}

// TestE2E_MakeFlags exercises the generator flags and error paths through the
// real binary. These checks only inspect generated files and generator behavior
// (they never compile the result), so they run against a dependency-free project
// to stay fast and network-independent.
func TestE2E_MakeFlags(t *testing.T) {
	dir := scaffoldNoDeps(t, "make_flags", []string{"postgres"})

	// --only generates just the handler, skipping its model/repository deps.
	runCrank(t, "", "make", "handler", "Coupon", "--only", "--project", dir)
	assertExists(t, dir, "internal/handler/coupon.go")
	assertNotExists(t, dir, "internal/model/coupon.go")
	assertNotExists(t, dir, "internal/repository/coupon.go")

	// --skip-migration suppresses the create-table migration even with postgres.
	runCrank(t, "", "make", "handler", "Promo", "code:string", "--skip-migration", "--project", dir)
	assertExists(t, dir, "internal/repository/promo.go")
	if n := globCount(t, dir, "migrations/*_create_promos.up.sql"); n != 0 {
		t.Errorf("--skip-migration should not create a migration, found %d", n)
	}

	// A primary-artifact conflict fails without --force and succeeds with it.
	runCrank(t, "", "make", "model", "Note", "--project", dir)
	if out, err := runCrankRaw(t, "", "make", "model", "Note", "--project", dir); err == nil {
		t.Errorf("expected a conflict error regenerating Note without --force:\n%s", out)
	} else if !strings.Contains(out, "already exists") {
		t.Errorf("expected an 'already exists' error, got:\n%s", out)
	}
	runCrank(t, "", "make", "model", "Note", "--force", "--project", dir)

	// Handler wiring is idempotent across regenerations.
	runCrank(t, "", "make", "handler", "Review", "--project", dir)
	runCrank(t, "", "make", "handler", "Review", "--force", "--project", dir)
	hub := readFile(t, dir, "internal/handler/handler.go")
	if n := strings.Count(hub, "h.reviews.Register(e)"); n != 1 {
		t.Errorf("expected exactly one review registration, got %d:\n%s", n, hub)
	}

	// Plural and multi-word inputs are normalized to a singular snake_case file.
	runCrank(t, "", "make", "model", "invoices", "--project", dir)
	assertExists(t, dir, "internal/model/invoice.go")
	runCrank(t, "", "make", "model", "OrderLine", "--project", dir)
	assertExists(t, dir, "internal/model/order_line.go")

	// Error surfaces: unknown kind, missing name, and unknown field type.
	if out, err := runCrankRaw(t, "", "make", "frobnicate", "Thing", "--project", dir); err == nil {
		t.Errorf("expected an unknown-kind error:\n%s", out)
	}
	if out, err := runCrankRaw(t, "", "make", "model", "--project", dir); err == nil {
		t.Errorf("expected a missing-name error:\n%s", out)
	}
	if out, err := runCrankRaw(t, "", "make", "model", "Widget", "size:gadget", "--project", dir); err == nil {
		t.Errorf("expected an unknown-field-type error:\n%s", out)
	}
}

// TestE2E_MakeTemporal exercises the Temporal workflow/activity generators end
// to end: it scaffolds a temporal-enabled project, generates a workflow and an
// activity (with tests), verifies they are auto-registered with the worker, and
// proves the whole project — example greeting workflow/activity, generated code
// and all generated tests — compiles, vets and passes against the real SDK.
func TestE2E_MakeTemporal(t *testing.T) {
	projectDir := scaffold(t, "make_temporal", []string{"temporal"})

	runCrank(t, "", "make", "workflow", "OrderFulfillment", "order_id:uuid", "--tests", "--project", projectDir)
	runCrank(t, "", "make", "activity", "ChargeCard", "amount:float", "--tests", "--project", projectDir)

	assertExists(t, projectDir, "internal/workflow/order_fulfillment.go")
	assertExists(t, projectDir, "internal/workflow/order_fulfillment_test.go")
	assertExists(t, projectDir, "internal/activity/charge_card.go")
	assertExists(t, projectDir, "internal/activity/charge_card_test.go")

	// Both are wired into the worker aggregator (alongside the shipped examples).
	worker := readFile(t, projectDir, "internal/temporal/worker.go")
	for _, want := range []string{
		"w.RegisterWorkflow(workflow.GreetingWorkflow)",
		"w.RegisterWorkflow(workflow.OrderFulfillmentWorkflow)",
		"w.RegisterActivity(activity.Greet)",
		"w.RegisterActivity(activity.ChargeCardActivity)",
	} {
		if !strings.Contains(worker, want) {
			t.Errorf("worker.go missing registration %q", want)
		}
	}

	compileProject(t, projectDir)
	runGo(t, projectDir, "test", "./internal/...")

	// The generators are gated behind the temporal feature.
	baseDir := scaffoldNoDeps(t, "no_temporal", []string{"base"})
	if out, err := runCrankRaw(t, "", "make", "workflow", "Foo", "--project", baseDir); err == nil {
		t.Errorf("expected `make workflow` to fail without the temporal feature:\n%s", out)
	}
}

// TestE2E_Add verifies that Add writes the new feature's files, injects
// config sections via markers (preserving existing content), and updates the manifest.
func TestE2E_Add(t *testing.T) {
	projectDir := scaffold(t, "svc_added", []string{"base"})

	for _, feature := range []string{"postgres", "auth"} {
		res, err := bootstrap.Add(bootstrap.GlobalRegistry, projectDir, feature)
		if err != nil {
			t.Fatalf("Add(%s): %v", feature, err)
		}
		if len(res.Dependencies) == 0 {
			t.Errorf("Add(%s): expected dependencies, got none", feature)
		}
		if !contains(res.Features, feature) {
			t.Errorf("Add(%s): feature not in result features %v", feature, res.Features)
		}
	}

	// Verify the manifest has all three features.
	manifest := readFile(t, projectDir, ".crank.yaml")
	for _, f := range []string{"base", "postgres", "auth"} {
		if !strings.Contains(manifest, "- "+f) {
			t.Errorf("manifest missing feature %q", f)
		}
	}

	// Verify new feature files were created.
	assertExists(t, projectDir, "internal/database/postgres.go")
	assertExists(t, projectDir, "internal/middleware/auth.go")
	assertExists(t, projectDir, "migrations/000001_init.up.sql")

	// Verify config sections were injected via markers.
	cfgGo := readFile(t, projectDir, "internal/config/config.go")
	if !strings.Contains(cfgGo, "DatabaseConfig") {
		t.Error("config.go after Add should contain DatabaseConfig")
	}
	if !strings.Contains(cfgGo, "JWTConfig") {
		t.Error("config.go after Add should contain JWTConfig")
	}
	// Verify markers are still present (so future Add calls work).
	if !strings.Contains(cfgGo, "// crank:config-fields") {
		t.Error("config.go should still have config-fields marker")
	}
	if !strings.Contains(cfgGo, "// crank:config-structs") {
		t.Error("config.go should still have config-structs marker")
	}
}

func contains(list []string, s string) bool {
	return slices.Contains(list, s)
}

func readFile(t *testing.T, dir, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(raw)
}

func assertExists(t *testing.T, dir, name string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, name)); os.IsNotExist(err) {
		t.Errorf("expected %s to exist", name)
	}
}

func assertNotExists(t *testing.T, dir, name string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
		t.Errorf("expected %s NOT to exist", name)
	}
}

// globCount returns the number of files matching the shell-style pattern
// (relative to dir). It fails the test on a malformed pattern.
func globCount(t *testing.T, dir, pattern string) int {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		t.Fatalf("glob %s: %v", pattern, err)
	}
	return len(matches)
}

// generateProject renders a project in-process (avoiding the binary's global
// tool auto-install side effects) and returns the generation result.
func generateProject(t *testing.T, name string, features []string) *bootstrap.Result {
	t.Helper()
	res, err := bootstrap.Generate(bootstrap.GlobalRegistry, bootstrap.Options{
		ProjectName: name,
		ModulePath:  "github.com/example/" + name,
		TargetDir:   t.TempDir(),
		Features:    features,
	})
	if err != nil {
		t.Fatalf("Generate(%s, %v): %v", name, features, err)
	}
	return res
}

// scaffold generates a project in-process and resolves its dependencies via
// `go get`, yielding a project that is ready to compile.
func scaffold(t *testing.T, name string, features []string) string {
	t.Helper()
	res := generateProject(t, name, features)
	if err := bootstrap.GoGet(res.ProjectDir, res.Dependencies); err != nil {
		t.Fatalf("resolve dependencies for %s: %v", name, err)
	}
	return res.ProjectDir
}

// scaffoldNoDeps generates a project in-process without resolving its Go module
// dependencies. It is used by tests that only inspect generated files or
// generator behavior (never compiling the result), keeping them fast and
// network-free.
func scaffoldNoDeps(t *testing.T, name string, features []string) string {
	t.Helper()
	return generateProject(t, name, features).ProjectDir
}

// compileProject runs `go build ./...` and `go vet ./...` in the generated
// project to verify the rendered templates produce valid, vet-clean Go code.
func compileProject(t *testing.T, projectDir string) {
	t.Helper()
	start := time.Now()
	runGo(t, projectDir, "build", "./...")
	runGo(t, projectDir, "vet", "./...")
	t.Logf("compiled %s in %s", filepath.Base(projectDir), time.Since(start).Round(time.Millisecond))
}

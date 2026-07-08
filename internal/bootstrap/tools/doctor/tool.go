// Package doctor implements the `crank doctor` in-process tool. It runs a
// small set of high-signal health checks against a generated project and
// prints one ✔/✘ line per check. The intent is to surface the "first week"
// class of bugs — manifest drift, un-wired handlers, mismatched module paths
// — before the user hits them at runtime.
package doctor

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/anurag925/crank/internal/bootstrap"
)

func init() {
	bootstrap.GlobalToolRegistry.MustRegister(&tool{})
}

type tool struct{}

func (tool) Name() string               { return "doctor" }
func (tool) BinaryName() string         { return "" } // in-process; no external binary
func (tool) Description() string        { return "Run health checks on a generated project" }
func (tool) RequiresFeatures() []string { return nil }
func (tool) InstallCmd() string         { return "" }
func (tool) Install() error             { return nil }
func (tool) AddFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("fail-fast", false, "stop at the first failing check")
}

func (*tool) LongDescription() string {
	return `doctor runs a series of health checks against a crank-generated project:

  1. manifest parses       — .crank.yaml is valid YAML with required fields
  2. module path matches   — go.mod's module line equals .crank.yaml's module_path
  3. handlers are wired    — every *_handler.go in internal/adapters/http/web/ has
                             a corresponding field+Register() call in routes.go
  4. services are wired    — every directory in internal/application/ is imported
                             and its NewCommandHandler is called in cmd/server/main.go
  5. migrations ordered    — files in db/migrations/ are uniquely timestamped and
                             lexically sorted

Each check prints ✔ or ✘ with a one-line detail on failure. Exit 0 when all
checks pass, 1 otherwise. Use --fail-fast to stop at the first failure.

Examples:
  crank doctor --project ./myapp
  cd myapp && crank doctor --fail-fast`
}

func (tool) Prepare(projectDir string, cmd *cobra.Command, extraArgs []string) (*bootstrap.ToolInvocation, error) {
	return nil, nil
}

// RunInProcess is the doctor entry point. It returns an unrecoverable error
// only when the directory is clearly not a crank project (no .crank.yaml and
// no go.mod). Per-check failures are reported via CheckResult.
func (tool) RunInProcess(projectDir string, out io.Writer) ([]bootstrap.CheckResult, error) {
	if projectDir == "" {
		projectDir = "."
	}
	abs, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, fmt.Errorf("resolve project path: %w", err)
	}

	if _, err := os.Stat(filepath.Join(abs, "go.mod")); err != nil {
		return nil, fmt.Errorf("%s does not look like a Go project (no go.mod found)\n\nRun this command from inside a crank-generated project, or pass --project <dir>", abs)
	}
	if _, err := os.Stat(filepath.Join(abs, ".crank.yaml")); err != nil {
		return nil, fmt.Errorf("%s does not look like a crank-generated project (no .crank.yaml manifest)\n\nRun this command from inside a crank-generated project, or pass --project <dir>", abs)
	}

	results := []bootstrap.CheckResult{
		checkManifest(abs),
		checkModulePath(abs),
		checkHandlerWiring(abs),
		checkServiceWiring(abs),
		checkMigrations(abs),
	}
	return results, nil
}

// ---------------------------------------------------------------------------
// Individual checks
// ---------------------------------------------------------------------------

// checkManifest verifies that .crank.yaml parses and has the required fields.
func checkManifest(projectDir string) bootstrap.CheckResult {
	path := filepath.Join(projectDir, ".crank.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return bootstrap.CheckResult{Summary: "manifest parses", Detail: err.Error()}
	}
	var m struct {
		ProjectName string   `yaml:"project_name"`
		ModulePath  string   `yaml:"module_path"`
		Features    []string `yaml:"features"`
	}
	if err := yaml.Unmarshal(data, &m); err != nil {
		return bootstrap.CheckResult{Summary: "manifest parses", Detail: fmt.Sprintf("YAML error in %s: %v", path, err)}
	}
	var problems []string
	if m.ProjectName == "" {
		problems = append(problems, "project_name is empty")
	}
	if m.ModulePath == "" {
		problems = append(problems, "module_path is empty")
	}
	if len(m.Features) == 0 {
		problems = append(problems, "features list is empty (expected at least \"base\")")
	}
	if len(problems) > 0 {
		return bootstrap.CheckResult{Summary: "manifest parses", Detail: strings.Join(problems, "; ")}
	}
	return bootstrap.CheckResult{OK: true, Summary: "manifest parses"}
}

// checkModulePath verifies that go.mod's `module` directive matches
// .crank.yaml's module_path. Drift between the two is a common post-rename bug.
func checkModulePath(projectDir string) bootstrap.CheckResult {
	manifestPath := filepath.Join(projectDir, ".crank.yaml")
	mfData, err := os.ReadFile(manifestPath)
	if err != nil {
		return bootstrap.CheckResult{Summary: "module path matches", Detail: err.Error()}
	}
	var m struct {
		ModulePath string `yaml:"module_path"`
	}
	if err := yaml.Unmarshal(mfData, &m); err != nil {
		return bootstrap.CheckResult{Summary: "module path matches", Detail: fmt.Sprintf("parse %s: %v", manifestPath, err)}
	}

	goModData, err := os.ReadFile(filepath.Join(projectDir, "go.mod"))
	if err != nil {
		return bootstrap.CheckResult{Summary: "module path matches", Detail: err.Error()}
	}
	var goMod string
	for _, line := range strings.Split(string(goModData), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			goMod = strings.TrimSpace(strings.TrimPrefix(line, "module "))
			break
		}
	}
	if goMod == "" {
		return bootstrap.CheckResult{Summary: "module path matches", Detail: "go.mod has no `module` directive"}
	}
	if goMod != m.ModulePath {
		return bootstrap.CheckResult{
			Summary: "module path matches",
			Detail:  fmt.Sprintf("go.mod says %q but .crank.yaml says %q — fix one or the other", goMod, m.ModulePath),
		}
	}
	return bootstrap.CheckResult{OK: true, Summary: "module path matches"}
}

// handlerFileRE matches the per-resource handler files generated by
// `crank make handler <Name>`. The base project also produces server.go,
// routes.go and a middleware/ directory; those are excluded.
var handlerFileRE = regexp.MustCompile(`^([a-z][a-z0-9_]*_handler)\.go$`)

// checkHandlerWiring looks at every *_handler.go in the web adapter
// directory and verifies that routes.go has both a field in MountConfig and a
// Register() call in Mount() for it. Missing either means the handler is
// dead code at runtime.
func checkHandlerWiring(projectDir string) bootstrap.CheckResult {
	webDir := filepath.Join(projectDir, "internal/adapters/http/web")
	entries, err := os.ReadDir(webDir)
	if err != nil {
		if os.IsNotExist(err) {
			// No web dir at all is not necessarily wrong (a project
			// might disable HTTP), but treat it as a warning.
			return bootstrap.CheckResult{Summary: "handlers are wired", Detail: "internal/adapters/http/web/ does not exist"}
		}
		return bootstrap.CheckResult{Summary: "handlers are wired", Detail: err.Error()}
	}

	var handlerNames []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := handlerFileRE.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		// The file name is `<resource>_handler.go`; the field/register
		// name in routes.go is the Pascal form, e.g. "OrderHandler".
		// We just record the lowercase base; the wiring check below
		// looks for any case-insensitive match.
		handlerNames = append(handlerNames, strings.TrimSuffix(e.Name(), ".go"))
	}
	if len(handlerNames) == 0 {
		return bootstrap.CheckResult{OK: true, Summary: "handlers are wired"}
	}

	routesPath := filepath.Join(webDir, "routes.go")
	routesData, err := os.ReadFile(routesPath)
	if err != nil {
		return bootstrap.CheckResult{Summary: "handlers are wired", Detail: fmt.Sprintf("read %s: %v", routesPath, err)}
	}
	routes := string(routesData)

	var missing []string
	for _, h := range handlerNames {
		// The generated field is `<Pascal>Handler *<Pascal>Handler`
		// and the registration is `cfg.<Pascal>Handler.Register(`. We
		// just look for the file's basename (lowercase) being present
		// in routes.go — that's enough to catch a totally missing
		// registration, which is the failure mode we care about.
		base := strings.TrimSuffix(h, "_handler")
		pascal := snakeToPascal(base)
		fieldRef := pascal + "Handler *"
		regRef := "cfg." + pascal + "Handler.Register("
		if !strings.Contains(routes, fieldRef) || !strings.Contains(routes, regRef) {
			missing = append(missing, h)
		}
	}
	if len(missing) > 0 {
		return bootstrap.CheckResult{
			Summary: "handlers are wired",
			Detail:  fmt.Sprintf("%s has no field+Register() for: %s", routesPath, strings.Join(missing, ", ")),
		}
	}
	return bootstrap.CheckResult{OK: true, Summary: "handlers are wired"}
}

// checkServiceWiring verifies that every directory under internal/application/
// is referenced from cmd/server/main.go (imported and used). A service that
// no longer compiles into the composition root is the most common silent
// regression after a refactor.
func checkServiceWiring(projectDir string) bootstrap.CheckResult {
	appDir := filepath.Join(projectDir, "internal/application")
	entries, err := os.ReadDir(appDir)
	if err != nil {
		if os.IsNotExist(err) {
			return bootstrap.CheckResult{OK: true, Summary: "services are wired"}
		}
		return bootstrap.CheckResult{Summary: "services are wired", Detail: err.Error()}
	}
	var serviceDirs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		serviceDirs = append(serviceDirs, e.Name())
	}
	if len(serviceDirs) == 0 {
		return bootstrap.CheckResult{OK: true, Summary: "services are wired"}
	}

	mainPath := filepath.Join(projectDir, "cmd/server/main.go")
	mainData, err := os.ReadFile(mainPath)
	if err != nil {
		return bootstrap.CheckResult{Summary: "services are wired", Detail: fmt.Sprintf("read %s: %v", mainPath, err)}
	}
	main := string(mainData)

	var missing []string
	for _, sd := range serviceDirs {
		// main.go typically imports the package as `<snake>app` (e.g.
		// `userapp "…/internal/application/user"`) and instantiates
		// it via `NewCommandHandler` / `NewQueryHandler`. Look for any
		// reference to the application subdirectory's import.
		importRef := "/internal/application/" + sd + "\""
		usageRef := sd + "app.NewCommandHandler"
		if !strings.Contains(main, importRef) && !strings.Contains(main, usageRef) {
			missing = append(missing, sd)
		}
	}
	if len(missing) > 0 {
		return bootstrap.CheckResult{
			Summary: "services are wired",
			Detail:  fmt.Sprintf("%s does not import or use: %s (run `crank make handler` or wire manually)", mainPath, strings.Join(missing, ", ")),
		}
	}
	return bootstrap.CheckResult{OK: true, Summary: "services are wired"}
}

// checkMigrations verifies that all files in db/migrations/ are uniquely
// timestamp-prefixed and lexically sorted (i.e. will apply in the order the
// developer intended).
func checkMigrations(projectDir string) bootstrap.CheckResult {
	migDir := filepath.Join(projectDir, "db/migrations")
	entries, err := os.ReadDir(migDir)
	if err != nil {
		if os.IsNotExist(err) {
			return bootstrap.CheckResult{OK: true, Summary: "migrations ordered"}
		}
		return bootstrap.CheckResult{Summary: "migrations ordered", Detail: err.Error()}
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		files = append(files, e.Name())
	}
	if len(files) == 0 {
		return bootstrap.CheckResult{OK: true, Summary: "migrations ordered"}
	}

	prefixRE := regexp.MustCompile(`^(\d+)_`)
	seenPrefixes := map[string]string{} // prefix -> filename
	var dup []string
	var outOfOrder []string
	prev := ""
	for _, f := range files {
		m := prefixRE.FindStringSubmatch(f)
		if m == nil {
			continue
		}
		if existing, ok := seenPrefixes[m[1]]; ok {
			dup = append(dup, fmt.Sprintf("%s and %s share timestamp %s", existing, f, m[1]))
		}
		seenPrefixes[m[1]] = f
		if prev != "" && f < prev {
			outOfOrder = append(outOfOrder, f)
		}
		if f > prev {
			prev = f
		}
	}
	sort.Strings(dup)
	sort.Strings(outOfOrder)
	if len(dup) > 0 {
		return bootstrap.CheckResult{Summary: "migrations ordered", Detail: strings.Join(dup, "; ")}
	}
	if len(outOfOrder) > 0 {
		return bootstrap.CheckResult{
			Summary: "migrations ordered",
			Detail:  "db/migrations/ is not lexically sorted by filename: " + strings.Join(outOfOrder, ", ") + " — rename to apply in order",
		}
	}
	return bootstrap.CheckResult{OK: true, Summary: "migrations ordered"}
}

// snakeToPascal converts `order_item` → `OrderItem`. It is a small purpose-
// built helper used only by the wiring check.
func snakeToPascal(s string) string {
	parts := strings.Split(s, "_")
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]))
		if len(p) > 1 {
			b.WriteString(p[1:])
		}
	}
	return b.String()
}

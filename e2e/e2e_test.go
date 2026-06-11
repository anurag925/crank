package e2e

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
)

// crankBin is the path to the crank binary built once in TestMain.
var crankBin string

// allFeatureNames lists every feature the application ships, used to validate
// the `crank list` output and to drive the "all features" compile test.
var allFeatureNames = []string{"base", "auth", "crypto", "postgres", "redis", "mongodb"}

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
	build.Env = append(os.Environ(), "GOTOOLCHAIN=local")
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
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// runGo runs a go subcommand inside dir and fails the test on error.
func runGo(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local")
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

// TestE2E_AddThenCompile generates a base project, adds postgres + auth via the
// generator's Add path, then compiles the result — exercising the `add` flow.
func TestE2E_AddThenCompile(t *testing.T) {
	projectDir := scaffold(t, "svc_added", []string{"base"})

	for _, feature := range []string{"postgres", "auth"} {
		res, err := bootstrap.Add(bootstrap.GlobalRegistry, projectDir, feature)
		if err != nil {
			t.Fatalf("Add(%s): %v", feature, err)
		}
		if len(res.Dependencies) > 0 {
			if err := bootstrap.GoGet(projectDir, res.Dependencies); err != nil {
				t.Fatalf("go get after adding %s: %v", feature, err)
			}
		}
	}
	compileProject(t, projectDir)
}

// scaffold generates a project in-process (avoiding the binary's global tool
// auto-install side effects) and resolves its dependencies via `go get`.
func scaffold(t *testing.T, name string, features []string) string {
	t.Helper()
	tmp := t.TempDir()
	res, err := bootstrap.Generate(bootstrap.GlobalRegistry, bootstrap.Options{
		ProjectName: name,
		ModulePath:  "github.com/example/" + name,
		TargetDir:   tmp,
		Features:    features,
	})
	if err != nil {
		t.Fatalf("Generate(%s, %v): %v", name, features, err)
	}
	if err := bootstrap.GoGet(res.ProjectDir, res.Dependencies); err != nil {
		t.Fatalf("resolve dependencies for %s: %v", name, err)
	}
	return res.ProjectDir
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

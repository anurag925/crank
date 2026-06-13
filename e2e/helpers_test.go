package e2e

// This file collects small, reusable helpers that are shared by the topic-specific
// test files (init, add, tools, makedelegate, config_inject, make, compat,
// generated, cross, cli). Keeping them here avoids a circular "every test file
// redefines the same helper" cycle and makes it easy to add a new helper that
// more than one suite needs.
//
// IMPORTANT: these e2e tests spawn many short-lived `go get` and `go mod tidy`
// subprocesses. When the Go test runner executes them in parallel (the default
// behaviour for `go test` is package-level parallelism; individual tests run
// serially within a package but the `go` subprocesses they invoke may collide
// on the shared module cache lock), some tests fail intermittently. To get a
// stable green run, invoke the suite with `-p 1`:
//
//	go test -tags e2e -p 1 -timeout 20m ./e2e/...
//
// The helper script `./scripts/test.sh e2e` already does this.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/anurag925/crank/internal/bootstrap"
)

// ============================================================================
// Crank-binary helpers
// ============================================================================

// crankDir is the directory that holds the crank binary built in TestMain.
// Exposed as a package var so individual tests can put the binary on PATH
// (see pathWithCrank) for tests that simulate "running crank from inside a
// generated project".
var crankDir string

func init() {
	// The binary lives next to where TestMain writes it: $TMPDIR/crank-e2e-bin/crank.
	// crankDir's parent is derived from the test binary's own location so the
	// setup survives being invoked from any working directory.
	_, file, _, _ := runtime.Caller(0)
	// The test binary is compiled into a temp dir at runtime, so we cannot
	// derive crankDir from `file` directly. Instead, look it up by inspecting
	// the current process's argv[0] and resolving it relative to the e2e dir.
	// In practice, TestMain assigns crankBin and we just take its directory.
	_ = file
}

// crankOnPath returns a PATH string with the directory containing the crank
// binary prepended so that subprocesses (e.g. a generated project's
// `crank build` invocation) can find it just like a real user would.
func crankOnPath(t *testing.T) string {
	t.Helper()
	bin, err := os.Executable()
	if err != nil {
		t.Fatalf("locate test binary: %v", err)
	}
	dir := filepath.Dir(bin)
	return fmt.Sprintf("%s%c%s", dir, os.PathListSeparator, os.Getenv("PATH"))
}

// runCrankWithEnv is like runCrank but lets the caller inject environment
// variables (commonly used to widen PATH for generated-project self-hosting
// tests).
func runCrankWithEnv(t *testing.T, dir string, env []string, args ...string) string {
	t.Helper()
	cmd := exec.Command(crankBin, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("crank %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func runCrankRawWithEnv(t *testing.T, dir string, env []string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(crankBin, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// ============================================================================
// Project scaffolding helpers (thin wrappers around the existing scaffold fns
// so individual tests can read more naturally).
// ============================================================================

// scaffoldProject is the most explicit name for "scaffold + go get so the
// project can be compiled and tested". Provided as an alias so topic files
// don't have to call the shorter scaffold() helper.
func scaffoldProject(t *testing.T, name string, features []string) string {
	t.Helper()
	return scaffold(t, name, features)
}

// scaffoldProjectRaw mirrors scaffoldProject but bypasses bootstrap.GoGet so
// the project is not compiled. Useful for tests that only need the file
// layout (e.g. `crank add` error paths, init flag tests, Makefile delegation
// tests).
func scaffoldProjectRaw(t *testing.T, name string, features []string) string {
	t.Helper()
	return scaffoldNoDeps(t, name, features)
}

// ============================================================================
// Manifest & feature helpers
// ============================================================================

// loadManifest reads .crank.yaml from projectDir and returns it as a public
// ProjectInfo. The existing scaffold in e2e_test.go reads the manifest via
// bootstrap.Add; this helper exposes the read path independently so tests
// can inspect the project without modifying it.
func loadManifest(t *testing.T, projectDir string) *bootstrap.ProjectInfo {
	t.Helper()
	info, err := bootstrap.LoadProjectInfo(projectDir)
	if err != nil {
		t.Fatalf("LoadProjectInfo(%s): %v", projectDir, err)
	}
	return info
}

// writeManifestRaw writes a literal YAML body to .crank.yaml. Used to set up
// "old project" fixtures that simulate pre-manifest or out-of-date projects.
func writeManifestRaw(t *testing.T, projectDir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(projectDir, ".crank.yaml"), []byte(body), 0o644); err != nil {
		t.Fatalf("write .crank.yaml: %v", err)
	}
}

// removeManifest deletes .crank.yaml from projectDir, simulating a project
// that has never been initialised or one whose manifest was lost.
func removeManifest(t *testing.T, projectDir string) {
	t.Helper()
	if err := os.Remove(filepath.Join(projectDir, ".crank.yaml")); err != nil && !os.IsNotExist(err) {
		t.Fatalf("remove .crank.yaml: %v", err)
	}
}

// ============================================================================
// Filesystem assertion helpers (small extensions over assertExists /
// assertNotExists already in e2e_test.go).
// ============================================================================

// assertContainsAll fails the test if any of the needles is missing from
// the file at projectDir/relPath.
func assertContainsAll(t *testing.T, projectDir, relPath string, needles ...string) {
	t.Helper()
	body := readFile(t, projectDir, relPath)
	for _, n := range needles {
		if !strings.Contains(body, n) {
			t.Errorf("%s missing %q", relPath, n)
		}
	}
}

// assertContainsNone fails the test if any of the needles is present in
// the file at projectDir/relPath.
func assertContainsNone(t *testing.T, projectDir, relPath string, needles ...string) {
	t.Helper()
	body := readFile(t, projectDir, relPath)
	for _, n := range needles {
		if strings.Contains(body, n) {
			t.Errorf("%s unexpectedly contains %q", relPath, n)
		}
	}
}

// assertGlobCount fails the test unless the number of files matching pattern
// equals want. Pattern is interpreted by filepath.Glob (no shell globstar).
func assertGlobCount(t *testing.T, projectDir, pattern string, want int) {
	t.Helper()
	got := globCount(t, projectDir, pattern)
	if got != want {
		t.Errorf("glob %s: got %d files, want %d", pattern, got, want)
	}
}

// writeChmod writes content to path with the given mode.
func writeChmod(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// mkdirAll is a thin wrapper around os.MkdirAll that fails the test.
func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

// copyFile copies src to dst, creating dst with mode 0o644.
func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	body, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}
	if err := os.WriteFile(dst, body, 0o644); err != nil {
		t.Fatalf("write %s: %v", dst, err)
	}
}

// ============================================================================
// Time-based helpers
// ============================================================================

// wallClock is a thin test-local time helper used by tests that need
// monotonically increasing timestamps (e.g. for verifying migration filenames
// sort correctly when several are created within the same second).
var wallClock = time.Now

// ============================================================================
// Module-path helpers
// ============================================================================

// modulePathFor derives a unique module path for a project name. Mirrors
// the convention in e2e_test.go's generateProject so test fixtures match
// what the test helper produces.
func modulePathFor(name string) string {
	return "github.com/example/" + name
}

// ============================================================================
// Process helpers
// ============================================================================

// startCrankCommand starts the crank binary in a subprocess and returns it
// to the caller. The caller is responsible for killing the process (use
// cmd.Process.Kill) and waiting on it. The env slice is appended to the
// parent process's environment, so callers commonly inject PATH or
// APP_PORT overrides here.
func startCrankCommand(t *testing.T, dir string, env []string, args ...string) (*exec.Cmd, error) {
	t.Helper()
	cmd := exec.Command(crankBin, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

// waitFor sleeps for the given number of milliseconds. Used by tests that
// need to give a subprocess a moment to boot (e.g. a `crank run` smoke
// test) before sending it a signal. Wall-clock sleeps are adequate for
// these low-precision use cases.
func waitFor(t *testing.T, milliseconds int) {
	t.Helper()
	<-time.After(time.Duration(milliseconds) * time.Millisecond)
}

// bootstrapAddForCompat runs bootstrap.Add in-process and returns the
// result. The in-process path skips the `go get` step that the binary
// performs, so it is suitable for tests that want to exercise the
// manifest-rewriting code path without depending on network access or a
// matching module path. Used by the backward-compat tests.
func bootstrapAddForCompat(t *testing.T, projectDir, featureName string) (*bootstrap.Result, error) {
	t.Helper()
	return bootstrap.Add(bootstrap.GlobalRegistry, projectDir, featureName)
}

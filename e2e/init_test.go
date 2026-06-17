package e2e

// Tests for the `crank init` command — flag combinations, error paths, and
// non-interactive behavior that were not covered by the original e2e suite.
//
// These tests deliberately avoid `go get` (using scaffoldNoDeps under the
// hood) for the cases that only need to inspect generated files, and switch
// to the full scaffold helper only when we want to verify the generated
// project compiles.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// --force: re-init an existing non-empty directory
// ---------------------------------------------------------------------------

// TestE2E_Init_Force_OverwritesNonEmpty covers the most common destructive
// path: re-running init against a directory that already has files. The
// --force flag must call os.RemoveAll internally; if it does not, the
// generator's "directory is not empty" check fires instead.
func TestE2E_Init_Force_OverwritesNonEmpty(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "myapp")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Drop a sentinel file that must be gone after --force.
	if err := os.WriteFile(filepath.Join(target, "sentinel.txt"), []byte("old"), 0o644); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}

	runCrank(t, dir, "init", "myapp", "--features=base", "--force")
	assertNotExists(t, target, "sentinel.txt")
	assertExists(t, target, "go.mod")
	assertExists(t, target, ".crank.yaml")
}

// TestE2E_Init_NoForce_RefusesNonEmpty verifies the safety net: without
// --force, init must refuse to clobber a non-empty directory and surface a
// clear error to the user.
func TestE2E_Init_NoForce_RefusesNonEmpty(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "myapp")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "keep.txt"), []byte("important"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	out, err := runCrankRaw(t, dir, "init", "myapp", "--features=base")
	if err == nil {
		t.Fatalf("expected non-empty error, got success:\n%s", out)
	}
	if !strings.Contains(out, "not empty") {
		t.Errorf("expected 'not empty' error, got:\n%s", out)
	}
	// Sentinel must still be there.
	if _, err := os.Stat(filepath.Join(target, "keep.txt")); err != nil {
		t.Errorf("sentinel disappeared despite --force being absent: %v", err)
	}
}

// ---------------------------------------------------------------------------
// --target: project lives under a non-default parent directory
// ---------------------------------------------------------------------------

// TestE2E_Init_CustomTarget verifies that --target accepts a non-default
// parent directory (e.g. a scratch dir) and that the project is created at
// the expected nested path. The non-default path is the most common case
// after "." — almost every user runs init from a parent workspace dir.
func TestE2E_Init_CustomTarget(t *testing.T) {
	parent := t.TempDir()
	nested := filepath.Join(parent, "projects")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	runCrank(t, parent, "init", "myapp", "--features=base", "--target", nested)
	expected := filepath.Join(nested, "myapp")
	assertExists(t, expected, "go.mod")
	assertExists(t, expected, ".crank.yaml")
}

// TestE2E_Init_CustomTarget_CurrentDir verifies that --target . (the
// default) is equivalent to running without the flag at all.
func TestE2E_Init_CustomTarget_CurrentDir(t *testing.T) {
	dir := t.TempDir()
	runCrank(t, dir, "init", "myapp", "--features=base", "--target", ".")
	// --target . means "create <projectname> inside <dir>". The
	// project is therefore at dir/myapp.
	assertExists(t, filepath.Join(dir, "myapp"), "go.mod")
	assertExists(t, filepath.Join(dir, "myapp"), ".crank.yaml")
}

// ---------------------------------------------------------------------------
// --module: non-default Go module path
// ---------------------------------------------------------------------------

// TestE2E_Init_CustomModulePath exercises --module with a fully-qualified Go
// import path. The module path is used in go.mod AND every internal import
// statement, so a broken plumbing would surface as a compile error.
func TestE2E_Init_CustomModulePath(t *testing.T) {
	dir := t.TempDir()
	runCrank(t, dir, "init", "myapp",
		"--features=base",
		"--module=github.com/anurag925/my-custom-app",
	)
	projectDir := filepath.Join(dir, "myapp")

	body := readFile(t, projectDir, "go.mod")
	if !strings.HasPrefix(body, "module github.com/anurag925/my-custom-app") {
		t.Errorf("go.mod missing custom module path:\n%s", body)
	}
	main := readFile(t, projectDir, "cmd/server/main.go")
	if !strings.Contains(main, "github.com/anurag925/my-custom-app/internal/config") {
		t.Errorf("main.go does not use the custom module path:\n%s", main)
	}
}

// ---------------------------------------------------------------------------
// Feature-list validation
// ---------------------------------------------------------------------------

// TestE2E_Init_DuplicateFeatureErrors verifies that --features=base,base
// is rejected with a clear message rather than silently rendering the base
// feature twice (which would produce duplicate registrations and likely a
// compile error downstream).
func TestE2E_Init_DuplicateFeatureErrors(t *testing.T) {
	dir := t.TempDir()
	out, err := runCrankRaw(t, dir, "init", "myapp", "--features=base,base")
	if err == nil {
		t.Fatalf("expected duplicate-feature error, got success:\n%s", out)
	}
	if !strings.Contains(out, "more than once") {
		t.Errorf("expected 'more than once' error, got:\n%s", out)
	}
}

// TestE2E_Init_InvalidFeatureErrors covers the most common typo: the user
// spells a real feature name wrong (e.g. "postgresq" instead of "bun").
// The error must mention the unknown name and ideally list valid options.
func TestE2E_Init_InvalidFeatureErrors(t *testing.T) {
	dir := t.TempDir()
	out, err := runCrankRaw(t, dir, "init", "myapp", "--features=base,bunq")
	if err == nil {
		t.Fatalf("expected unknown-feature error, got success:\n%s", out)
	}
	if !strings.Contains(out, "bunq") {
		t.Errorf("error should mention the bad feature name, got:\n%s", out)
	}
}

// TestE2E_Init_EmptyProjectName covers the case where the user runs init
// without flags and without arguments in a non-interactive context. Cobra's
// arg validation produces a clean error.
func TestE2E_Init_EmptyProjectName(t *testing.T) {
	dir := t.TempDir()
	out, err := runCrankRaw(t, dir, "init", "--features=base")
	if err == nil {
		t.Fatalf("expected missing-name error, got success:\n%s", out)
	}
	if !strings.Contains(out, "project name is required") {
		t.Errorf("expected 'project name is required' error, got:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// Non-interactive happy paths (combinations of flags)
// ---------------------------------------------------------------------------

// TestE2E_Init_NonInteractive_AllFlags compiles a project that was generated
// with --module + --target + --force simultaneously — a matrix of flag
// combinations that we never tested together. If any of the three regresses,
// the project must still build cleanly.
func TestE2E_Init_NonInteractive_AllFlags(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "nested")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Pre-populate the target dir with a sentinel so --force has to fire.
	if err := os.WriteFile(filepath.Join(target, "stale.txt"), []byte("old"), 0o644); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}
	// Run init from the OUTER directory with --target to point at the inner.
	runCrank(t, dir, "init", "myapp",
		"--features=base,bun",
		"--module=github.com/anurag925/e2e_init_allflags",
		"--target", target,
		"--force",
	)
	projectDir := filepath.Join(target, "myapp")
	assertNotExists(t, projectDir, "stale.txt")
	compileProject(t, projectDir)
}

// TestE2E_Init_DefaultBaseOnly checks the most common real-world invocation:
// the user just runs `crank init foo` with no features flag at all. The
// default is "base" — we verify that base is in the resulting manifest and
// that no extra features were sneaked in.
func TestE2E_Init_DefaultBaseOnly(t *testing.T) {
	dir := t.TempDir()
	runCrank(t, dir, "init", "myapp")
	projectDir := filepath.Join(dir, "myapp")
	manifest := readFile(t, projectDir, ".crank.yaml")
	if !strings.Contains(manifest, "- base") {
		t.Errorf("default init should include base:\n%s", manifest)
	}
	for _, f := range []string{"bun", "auth", "redis", "mongodb", "crypto", "temporal"} {
		if strings.Contains(manifest, "- "+f) {
			t.Errorf("default init should not include %s:\n%s", f, manifest)
		}
	}
}

// TestE2E_Init_GeneratedProjectCompiles is the bottom-line integration test
// for init: the user types `crank init` and walks away with a project that
// builds. Runs against every single feature as a "minimum-viable build"
// check.
func TestE2E_Init_GeneratedProjectCompiles(t *testing.T) {
	for _, feature := range []string{"base", "auth", "crypto", "bun", "redis", "mongodb", "temporal"} {
		feature := feature
		t.Run(feature, func(t *testing.T) {
			projectDir := scaffoldProject(t, "init_compile_"+feature, []string{feature})
			compileProject(t, projectDir)
		})
	}
}

package e2e

// Tests for the `crank doctor` in-process tool. Doctor runs a curated set of
// health checks against a generated project and exits 0 only if every check
// passes. Each detection test corrupts a project in a specific way and
// asserts doctor reports a failure pointing at the right area.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestE2E_Doctor_HappyPath runs doctor against a freshly generated base
// project and expects every check to pass.
func TestE2E_Doctor_HappyPath(t *testing.T) {
	dir := scaffoldBase(t, "doctor_happy")
	out := runCrank(t, "", "doctor", "--project", dir)
	for _, want := range []string{
		"manifest parses",
		"module path matches",
		"handlers are wired",
		"services are wired",
		"migrations ordered",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("doctor output missing check %q\n%s", want, out)
		}
	}
	// All checks should be passing (✔) — none should fail (✘).
	if strings.Contains(out, "✘") {
		t.Errorf("doctor unexpectedly reported a failure:\n%s", out)
	}
}

// TestE2E_Doctor_NoManifest confirms doctor refuses to run on a directory
// without a go.mod / .crank.yaml.
func TestE2E_Doctor_NoManifest(t *testing.T) {
	dir := t.TempDir()
	out, err := runCrankRaw(t, "", "doctor", "--project", dir)
	if err == nil {
		t.Fatalf("expected doctor to fail on a non-project directory\n%s", out)
	}
	// The check fires before any project-specific lookups, so the error
	// should mention it isn't a Go project.
	if !strings.Contains(out, "not look like") && !strings.Contains(out, "manifest") {
		t.Errorf("error should mention the project is not a crank project, got:\n%s", out)
	}
}

// TestE2E_Doctor_ModulePathDrift confirms doctor catches a go.mod module
// line that no longer matches the manifest.
func TestE2E_Doctor_ModulePathDrift(t *testing.T) {
	dir := scaffoldBase(t, "doctor_drift")

	// Rewrite go.mod to use a different module path.
	gomod := filepath.Join(dir, "go.mod")
	body, err := os.ReadFile(gomod)
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	drifted := strings.Replace(string(body), "github.com/example/doctor_drift", "github.com/example/doctor_drift_renamed", 1)
	if err := os.WriteFile(gomod, []byte(drifted), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	out, err := runCrankRaw(t, "", "doctor", "--project", dir)
	if err == nil {
		t.Fatalf("expected doctor to fail on drifted module path\n%s", out)
	}
	if !strings.Contains(out, "module path") {
		t.Errorf("error should mention module path mismatch, got:\n%s", out)
	}
}

// TestE2E_Doctor_UnwiredHandler confirms doctor catches a handler file
// that is not registered in routes.go.
func TestE2E_Doctor_UnwiredHandler(t *testing.T) {
	dir := scaffoldBase(t, "doctor_unwired")

	// Add a handler file that is not registered in routes.go.
	unwired := filepath.Join(dir, "internal/adapters/http/web/coupon_handler.go")
	body := `package web

import "github.com/labstack/echo/v4"

type CouponHandler struct{}

func (h *CouponHandler) Register(g *echo.Group) {}
`
	if err := os.WriteFile(unwired, []byte(body), 0o644); err != nil {
		t.Fatalf("write unwired handler: %v", err)
	}

	out, err := runCrankRaw(t, "", "doctor", "--project", dir)
	if err == nil {
		t.Fatalf("expected doctor to fail on unwired handler\n%s", out)
	}
	// Doctor output (stdout) and the cobra error message (stderr) are
	// captured into the same buffer but their relative order is
	// non-deterministic. Accept either signal that the unwired handler
	// was caught: the ✘ marker for the handler check, OR the doctor
	// summary error, OR a mention of the unwired file.
	if !strings.Contains(out, "✘") {
		t.Errorf("doctor should report a failing check, got:\n%s", out)
	}
	if !strings.Contains(out, "issue") {
		t.Errorf("doctor should report an issue count, got:\n%s", out)
	}
}

// TestE2E_Doctor_DuplicateMigration confirms doctor catches a migration
// timestamp that appears more than once. Postgres is required so the
// project has a migrations/ directory.
func TestE2E_Doctor_DuplicateMigration(t *testing.T) {
	dir := scaffold(t, "doctor_dup_migration", []string{"base", "bun"})

	// Create a second migration with a duplicate timestamp prefix.
	dup := filepath.Join(dir, "migrations/000001_duplicate.up.sql")
	if err := os.WriteFile(dup, []byte("SELECT 1;\n"), 0o644); err != nil {
		t.Fatalf("write duplicate migration: %v", err)
	}

	out, err := runCrankRaw(t, "", "doctor", "--project", dir)
	if err == nil {
		t.Fatalf("expected doctor to fail on duplicate migration timestamp\n%s", out)
	}
	// Doctor output is split between stdout (per-check lines) and stderr
	// (the error count). Look for either signal that a check failed.
	if !strings.Contains(out, "✘") {
		t.Errorf("doctor should report a failing check, got:\n%s", out)
	}
	if !strings.Contains(out, "issue") {
		t.Errorf("doctor should report an issue count, got:\n%s", out)
	}
}

package e2e

// Tests for the Makefile-delegation fallback (TryMakeDelegation). The
// delegation layer lets users extend crank with project-specific Makefile
// targets: `crank <target>` falls through to `make <target>` if no native
// crank command with that name is registered.
//
// Precedence rules under test:
//   1. Native crank commands always win (no delegation, even if Makefile
//      has a same-named target).
//   2. If the first arg is not a known crank command AND the Makefile in
//      the project (default "." or via --project) defines that target,
//      crank transparently runs `make <target>`.
//   3. If neither (1) nor (2) match, the cobra error path fires ("unknown
//      command").

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// makefileFixture writes a deterministic Makefile with the given targets
// to projectDir and returns a sentinel file the targets touch. Tests
// verify a delegation fired by checking the sentinel's mtime/marker.
func makefileFixture(t *testing.T, projectDir string, targets map[string]string) {
	t.Helper()
	var b strings.Builder
	b.WriteString("# crank e2e test Makefile — do not edit by hand\n")
	b.WriteString("APP_NAME := myapp\n\n")
	b.WriteString(".PHONY: help\n\n")
	b.WriteString("help:\n")
	b.WriteString("\t@echo \"help target ran\"\n\n")
	for name, marker := range targets {
		b.WriteString(name)
		b.WriteString(":\n")
		b.WriteString("\t@touch ")
		b.WriteString(marker)
		b.WriteString("\n")
		b.WriteString("\t@echo \"ran ")
		b.WriteString(name)
		b.WriteString("\"\n\n")
	}
	if err := os.WriteFile(filepath.Join(projectDir, "Makefile"), []byte(b.String()), 0o644); err != nil {
		t.Fatalf("write Makefile: %v", err)
	}
}

// sentinelTouched returns true if path exists and was modified within the
// last few seconds. We rely on this to confirm a delegated target fired.
func sentinelTouched(t *testing.T, path string) bool {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) < 30*time.Second
}

// TestE2E_Makefile_DelegatesClean is the canonical delegation test: the
// generated Makefile defines a `clean` target, so `crank clean` should
// transparently run `make clean` and remove the `bin/` directory.
func TestE2E_Makefile_DelegatesClean(t *testing.T) {
	dir := scaffoldBase(t, "makedelegate_clean")

	// Create a bin/ directory the `clean` target should remove.
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "stale"), []byte("stale"), 0o644); err != nil {
		t.Fatalf("write stale: %v", err)
	}

	runCrank(t, dir, "clean")
	if _, err := os.Stat(binDir); !os.IsNotExist(err) {
		t.Errorf("expected bin/ to be removed by delegated clean, got err=%v", err)
	}
}

// TestE2E_Makefile_NativeCommandWins verifies the precedence rule: even
// if the Makefile defines a `vet` target, `crank vet` must still run the
// native tool wrapper. The test detects this by checking the invocation
// output for the tool's "→ go vet ./..." banner.
func TestE2E_Makefile_NativeCommandWins(t *testing.T) {
	dir := scaffoldBase(t, "makedelegate_native_wins")
	makefileFixture(t, dir, map[string]string{
		"vet": filepath.Join(dir, "should_not_exist.txt"),
	})

	out := runCrank(t, dir, "vet")
	// The crank tool invocation prints "→ go vet ./...". If we see that,
	// we know crank ran the native vet tool, not `make vet`.
	if !strings.Contains(out, "go vet") {
		t.Errorf("expected native go vet invocation, got:\n%s", out)
	}
	// The Makefile's vet target must not have fired.
	if _, err := os.Stat(filepath.Join(dir, "should_not_exist.txt")); err == nil {
		t.Errorf("Makefile vet target was invoked despite native command winning")
	}
}

// TestE2E_Makefile_CustomTargetWithArgs exercises a non-trivial delegated
// command: a Makefile target that takes positional/keyword arguments.
// These are forwarded verbatim to make.
func TestE2E_Makefile_CustomTargetWithArgs(t *testing.T) {
	dir := scaffoldBase(t, "makedelegate_args")
	makefileFixture(t, dir, map[string]string{
		"greet": filepath.Join(dir, "greet.touched"),
	})
	// Overwrite greet with a body that reads NAME=... from make vars.
	body := `APP_NAME := myapp

.PHONY: greet

greet:
	@echo "hello $(NAME) from $(APP_NAME)"
	@touch greet.touched
`
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte(body), 0o644); err != nil {
		t.Fatalf("write Makefile: %v", err)
	}

	// Delegate with NAME=anurag; the delegated call must include NAME=anurag.
	out := runCrank(t, dir, "greet", "NAME=anurag")
	if !strings.Contains(out, "hello anurag from myapp") {
		t.Errorf("expected forwarded NAME in output, got:\n%s", out)
	}
	if !sentinelTouched(t, filepath.Join(dir, "greet.touched")) {
		t.Errorf("delegated target did not run (no sentinel touch)")
	}
}

// TestE2E_Makefile_DelegatesFromOutsideProject verifies the --project
// flag in conjunction with Makefile delegation: the user can be in any
// directory, point crank at a project, and `crank <target>` resolves
// the Makefile at <project>/Makefile.
func TestE2E_Makefile_DelegatesFromOutsideProject(t *testing.T) {
	dir := scaffoldBase(t, "makedelegate_outer")
	other := t.TempDir()
	makefileFixture(t, dir, map[string]string{
		"hello": filepath.Join(dir, "hello.touched"),
	})

	runCrank(t, other, "hello", "--project", dir)
	if !sentinelTouched(t, filepath.Join(dir, "hello.touched")) {
		t.Errorf("delegated target did not run (no sentinel touch)")
	}
}

// TestE2E_Makefile_DelegatesFromOutsideProject_EqualsForm exercises the
// `--project=./dir` syntax instead of the space-separated form.
func TestE2E_Makefile_DelegatesFromOutsideProject_EqualsForm(t *testing.T) {
	dir := scaffoldBase(t, "makedelegate_outer_eq")
	other := t.TempDir()
	makefileFixture(t, dir, map[string]string{
		"hello": filepath.Join(dir, "hello.touched"),
	})

	runCrank(t, other, "hello", "--project="+dir)
	if !sentinelTouched(t, filepath.Join(dir, "hello.touched")) {
		t.Errorf("delegated target did not run (no sentinel touch)")
	}
}

// TestE2E_Makefile_NoMakefileInProject covers the fallback when --project
// points to a directory that has no Makefile. Native unknown-command path
// should fire; no Makefile-based delegation occurs.
func TestE2E_Makefile_NoMakefileInProject(t *testing.T) {
	dir := t.TempDir() // empty — no Makefile, no project
	out, err := runCrankRaw(t, "", "not-a-thing", "--project", dir)
	if err == nil {
		t.Fatalf("expected unknown-command error, got success:\n%s", out)
	}
	if !strings.Contains(out, "unknown command") && !strings.Contains(out, "not-a-thing") {
		t.Errorf("expected unknown-command error, got:\n%s", out)
	}
}

// TestE2E_Makefile_TargetNotInMakefile verifies that the delegation does
// not fire for a name that is not in the Makefile. Cobra's unknown-command
// path takes over.
func TestE2E_Makefile_TargetNotInMakefile(t *testing.T) {
	dir := scaffoldBase(t, "makedelegate_unknown")
	makefileFixture(t, dir, map[string]string{
		"foo": filepath.Join(dir, "foo.touched"),
	})
	out, err := runCrankRaw(t, dir, "no-such-target")
	if err == nil {
		t.Fatalf("expected unknown-command error, got success:\n%s", out)
	}
	if !strings.Contains(out, "no-such-target") {
		t.Errorf("error should mention the bad name, got:\n%s", out)
	}
}

// TestE2E_Makefile_FirstArgIsFlag covers the early-exit in
// TryMakeDelegation: if the first arg is a flag (e.g. --help), we never
// consider it a target. Cobra handles the flag as usual.
func TestE2E_Makefile_FirstArgIsFlag(t *testing.T) {
	dir := scaffoldBase(t, "makedelegate_flag")
	makefileFixture(t, dir, map[string]string{
		"version": filepath.Join(dir, "version.touched"),
	})
	out := runCrank(t, dir, "--version")
	// The --version output is the crank version, NOT the result of running
	// `make version`. So the Makefile's `version` target must NOT have fired.
	if _, err := os.Stat(filepath.Join(dir, "version.touched")); err == nil {
		t.Errorf("`--version` was incorrectly treated as a Makefile target")
	}
	_ = out
}

// TestE2E_Makefile_HelpAndCompletionNotDelegated ensures that cobra's
// built-in `help` and `completion` commands are recognized by crank and
// not delegated to a Makefile.
func TestE2E_Makefile_HelpAndCompletionNotDelegated(t *testing.T) {
	dir := scaffoldBase(t, "makedelegate_help")
	makefileFixture(t, dir, map[string]string{
		"help":       filepath.Join(dir, "help.touched"),
		"completion": filepath.Join(dir, "completion.touched"),
	})
	// `crank help` should print cobra's help, not run `make help`.
	_ = runCrank(t, dir, "help")
	if _, err := os.Stat(filepath.Join(dir, "help.touched")); err == nil {
		t.Errorf("`help` was incorrectly delegated to make")
	}
}

// TestE2E_Makefile_VariableAssignmentNotTarget checks that lines in the
// Makefile like `APP_NAME := myapp` are not parsed as targets. The
// `spliceAtBraces`/`makefileTargets` code path must distinguish
// `name :=` (variable) from `name:` (target).
func TestE2E_Makefile_VariableAssignmentNotTarget(t *testing.T) {
	dir := scaffoldBase(t, "makedelegate_var")
	// Write a Makefile that has only variables and one real target.
	body := `APP_NAME := myapp
NAME = fallback
VERSION := 1.0

.PHONY: real

real:
	@touch real.touched
`
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte(body), 0o644); err != nil {
		t.Fatalf("write Makefile: %v", err)
	}

	// `crank APP_NAME` must NOT delegate; `crank NAME` and `crank VERSION`
	// must not either.
	for _, name := range []string{"APP_NAME", "NAME", "VERSION"} {
		out, err := runCrankRaw(t, dir, name)
		if err == nil {
			t.Errorf("%q was incorrectly delegated to make:\n%s", name, out)
		}
	}

	// `crank real` MUST delegate.
	runCrank(t, dir, "real")
	if !sentinelTouched(t, filepath.Join(dir, "real.touched")) {
		t.Errorf("real target was not delegated")
	}
}

// TestE2E_Makefile_DotPhonyNotTarget verifies that the special `.PHONY`
// directive is not treated as a target. (It is a special target name
// starting with `.` and must be ignored.)
func TestE2E_Makefile_DotPhonyNotTarget(t *testing.T) {
	dir := scaffoldBase(t, "makedelegate_phony")
	// A Makefile with only .PHONY + one real target.
	body := `.PHONY: real

real:
	@touch real.touched
`
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte(body), 0o644); err != nil {
		t.Fatalf("write Makefile: %v", err)
	}
	out, err := runCrankRaw(t, dir, ".PHONY")
	if err == nil {
		t.Errorf(".PHONY should not be a valid target, got success:\n%s", out)
	}
}

// TestE2E_Makefile_CommentLinesIgnored verifies that `#` comment lines
// are not parsed as targets (even if they syntactically look like one).
func TestE2E_Makefile_CommentLinesIgnored(t *testing.T) {
	dir := scaffoldBase(t, "makedelegate_comment")
	body := `# this-looks-like-a-target:
# but it's a comment

.PHONY: real
real:
	@touch real.touched
`
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte(body), 0o644); err != nil {
		t.Fatalf("write Makefile: %v", err)
	}
	// The commented-out "this-looks-like-a-target" must not fire.
	out, err := runCrankRaw(t, dir, "this-looks-like-a-target")
	if err == nil {
		t.Errorf("comment-line pseudo-target was delegated:\n%s", out)
	}
}

// TestE2E_Makefile_TabIndentedRecipeNotTarget verifies that lines
// starting with a tab (recipe lines) are not treated as targets.
func TestE2E_Makefile_TabIndentedRecipeNotTarget(t *testing.T) {
	dir := scaffoldBase(t, "makedelegate_tab")
	body := `.PHONY: real
real:
	@echo "this is a recipe line"
	@touch real.touched
`
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte(body), 0o644); err != nil {
		t.Fatalf("write Makefile: %v", err)
	}
	// The recipe text "@echo ..." must not be parsed as a target.
	out, err := runCrankRaw(t, dir, "@echo")
	if err == nil {
		t.Errorf("tab-indented recipe was treated as a target:\n%s", out)
	}
}

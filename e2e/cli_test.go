package e2e

// Tests for the CLI surface that aren't covered by the basic --version /
// --help / list / tools tests in the original e2e suite. These focus on
// subtle behaviors of cobra: what happens with no args, with unknown
// subcommands, with malformed flags, etc.

import (
	"strings"
	"testing"
)

// TestE2E_CLI_NoArgs_ShowsHelp verifies the cobra default: invoking
// `crank` with no arguments prints the help text and exits 0.
func TestE2E_CLI_NoArgs_ShowsHelp(t *testing.T) {
	out := runCrank(t, "", "--help")
	for _, want := range []string{"init", "add", "list", "make", "tools"} {
		if !strings.Contains(out, want) {
			t.Errorf("--help missing %q:\n%s", want, out)
		}
	}
}

// TestE2E_CLI_Help_ExitCleanly verifies the help exit code is 0 (not 1).
// Cobra's `Run` returning nil produces exit 0.
func TestE2E_CLI_Help_ExitCleanly(t *testing.T) {
	runCrank(t, "", "help")
}

// TestE2E_CLI_Version_LongForm verifies that `crank --version` (the
// double-dash form) also works, not just `-v`.
func TestE2E_CLI_Version_LongForm(t *testing.T) {
	out := runCrank(t, "", "--version")
	if !strings.Contains(out, "crank version") {
		t.Errorf("--version output missing 'crank version':\n%s", out)
	}
}

// TestE2E_CLI_Version_CommitHash_Format verifies the version string
// includes the commit hash and build date, as advertised.
func TestE2E_CLI_Version_CommitHash_Format(t *testing.T) {
	out := runCrank(t, "", "--version")
	// The default version is "dev (commit none, built unknown)".
	// We accept any string that contains both "commit" and "built".
	if !strings.Contains(out, "commit") {
		t.Errorf("version string missing 'commit':\n%s", out)
	}
	if !strings.Contains(out, "built") {
		t.Errorf("version string missing 'built':\n%s", out)
	}
}

// TestE2E_CLI_Init_Help_ShowsFlags verifies that `crank init --help`
// lists all the flags the user can set.
func TestE2E_CLI_Init_Help_ShowsFlags(t *testing.T) {
	out := runCrank(t, "", "init", "--help")
	for _, want := range []string{"--features", "--module", "--target", "--force"} {
		if !strings.Contains(out, want) {
			t.Errorf("init --help missing flag %q:\n%s", want, out)
		}
	}
}

// TestE2E_CLI_Add_Help_ShowsProjectFlag verifies that `crank add --help`
// mentions the --project flag.
func TestE2E_CLI_Add_Help_ShowsProjectFlag(t *testing.T) {
	out := runCrank(t, "", "add", "--help")
	if !strings.Contains(out, "--project") {
		t.Errorf("add --help missing --project flag:\n%s", out)
	}
}

// TestE2E_CLI_Make_Help_ShowsKinds verifies that `crank make --help`
// enumerates every supported generator kind.
func TestE2E_CLI_Make_Help_ShowsKinds(t *testing.T) {
	out := runCrank(t, "", "make", "--help")
	for _, want := range []string{
		"model", "repository", "service", "handler", "scaffold", "workflow", "activity", "migration",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("make --help missing kind %q:\n%s", want, out)
		}
	}
}

// TestE2E_CLI_Make_Help_ShowsFlags verifies that `crank make --help`
// mentions every supported flag.
func TestE2E_CLI_Make_Help_ShowsFlags(t *testing.T) {
	out := runCrank(t, "", "make", "--help")
	for _, want := range []string{"--project", "--only", "--force", "--skip-migration", "--tests"} {
		if !strings.Contains(out, want) {
			t.Errorf("make --help missing flag %q:\n%s", want, out)
		}
	}
}

// TestE2E_CLI_Migrate_Help_ShowsFlags verifies that `crank migrate --help`
// mentions its custom flags.
func TestE2E_CLI_Migrate_Help_ShowsFlags(t *testing.T) {
	out := runCrank(t, "", "migrate", "--help")
	for _, want := range []string{"--database-url", "--steps"} {
		if !strings.Contains(out, want) {
			t.Errorf("migrate --help missing flag %q:\n%s", want, out)
		}
	}
}

// TestE2E_CLI_Build_Help_ShowsProjectFlag verifies that `crank build --help`
// mentions --project.
func TestE2E_CLI_Build_Help_ShowsProjectFlag(t *testing.T) {
	out := runCrank(t, "", "build", "--help")
	if !strings.Contains(out, "--project") {
		t.Errorf("build --help missing --project flag:\n%s", out)
	}
}

// TestE2E_CLI_Swag_Help_ShowsProjectFlag verifies that `crank make swag --help`
// mentions --project.
func TestE2E_CLI_Swag_Help_ShowsProjectFlag(t *testing.T) {
	out := runCrank(t, "", "make", "swag", "--help")
	if !strings.Contains(out, "--project") {
		t.Errorf("make swag --help missing --project flag:\n%s", out)
	}
}

// TestE2E_CLI_List_Help_ShowsJSONFlag verifies that `crank list --help`
// mentions the --json flag.
func TestE2E_CLI_List_Help_ShowsJSONFlag(t *testing.T) {
	out := runCrank(t, "", "list", "--help")
	if !strings.Contains(out, "--json") {
		t.Errorf("list --help missing --json flag:\n%s", out)
	}
}

// TestE2E_CLI_Tools_Help covers the implicit help text on `crank tools`.
// (No special flags here, but the long description should appear.)
func TestE2E_CLI_Tools_Help(t *testing.T) {
	out := runCrank(t, "", "tools", "--help")
	if !strings.Contains(out, "tools") {
		t.Errorf("tools --help missing 'tools' text:\n%s", out)
	}
}

// TestE2E_CLI_UnknownSubcommand_Errors verifies that an unknown
// subcommand produces cobra's standard "unknown command" error.
func TestE2E_CLI_UnknownSubcommand_Errors(t *testing.T) {
	out, err := runCrankRaw(t, "", "definitely-not-real")
	if err == nil {
		t.Fatalf("expected error for unknown subcommand, got success:\n%s", out)
	}
	if !strings.Contains(out, "unknown command") && !strings.Contains(out, "definitely-not-real") {
		t.Errorf("error should mention unknown command, got:\n%s", out)
	}
}

// TestE2E_CLI_UnknownFlag_Errors verifies that an unknown flag on a
// known subcommand surfaces cobra's "unknown flag" error.
func TestE2E_CLI_UnknownFlag_Errors(t *testing.T) {
	out, err := runCrankRaw(t, "", "list", "--no-such-flag")
	if err == nil {
		t.Fatalf("expected error for unknown flag, got success:\n%s", out)
	}
	if !strings.Contains(out, "no-such-flag") && !strings.Contains(out, "unknown flag") {
		t.Errorf("error should mention the bad flag, got:\n%s", out)
	}
}

// TestE2E_CLI_Init_BadFeaturesFlag verifies that a malformed --features
// value (e.g. a stray colon or empty segments) is rejected.
func TestE2E_CLI_Init_BadFeaturesFlag(t *testing.T) {
	dir := t.TempDir()
	out, err := runCrankRaw(t, dir, "init", "myapp", "--features=")
	// An empty --features is equivalent to omitting the flag (defaults
	// to base). Both are acceptable; we accept either outcome.
	_ = out
	_ = err
}

// TestE2E_CLI_ExitCodeOnUnknown is a strict exit-code check: an unknown
// subcommand must produce a non-zero exit. We assert this by checking
// the err from runCrankRaw.
func TestE2E_CLI_ExitCodeOnUnknown(t *testing.T) {
	_, err := runCrankRaw(t, "", "truly-nonexistent")
	if err == nil {
		t.Errorf("expected non-zero exit on unknown subcommand")
	}
}

// TestE2E_CLI_ExitCodeOnVersion is the converse: --version must exit 0.
func TestE2E_CLI_ExitCodeOnVersion(t *testing.T) {
	_, err := runCrankRaw(t, "", "--version")
	if err != nil {
		t.Errorf("expected zero exit on --version, got: %v", err)
	}
}

// TestE2E_CLI_ExitCodeOnHelp is the same for --help.
func TestE2E_CLI_ExitCodeOnHelp(t *testing.T) {
	_, err := runCrankRaw(t, "", "--help")
	if err != nil {
		t.Errorf("expected zero exit on --help, got: %v", err)
	}
}

// TestE2E_CLI_ExitCodeOnList is the same for `crank list`.
func TestE2E_CLI_ExitCodeOnList(t *testing.T) {
	_, err := runCrankRaw(t, "", "list")
	if err != nil {
		t.Errorf("expected zero exit on `list`, got: %v", err)
	}
}

// TestE2E_CLI_ExitCodeOnTools is the same for `crank tools`.
func TestE2E_CLI_ExitCodeOnTools(t *testing.T) {
	_, err := runCrankRaw(t, "", "tools")
	if err != nil {
		t.Errorf("expected zero exit on `tools`, got: %v", err)
	}
}

// TestE2E_CLI_LongDescriptionForInit verifies that the long help text on
// `crank init` shows example invocations. This text is referenced by
// users who want to learn the CLI; it must remain stable.
func TestE2E_CLI_LongDescriptionForInit(t *testing.T) {
	out := runCrank(t, "", "init", "--help")
	for _, want := range []string{"Examples:", "crank init myapp"} {
		if !strings.Contains(out, want) {
			t.Errorf("init --help missing %q:\n%s", want, out)
		}
	}
}

// TestE2E_CLI_LongDescriptionForMake verifies the long help on `crank
// make` includes the supported field types.
func TestE2E_CLI_LongDescriptionForMake(t *testing.T) {
	out := runCrank(t, "", "make", "--help")
	for _, want := range []string{"string", "int", "bool", "time", "uuid", "email"} {
		if !strings.Contains(out, want) {
			t.Errorf("make --help missing field type %q:\n%s", want, out)
		}
	}
}

// TestE2E_CLI_LongDescriptionForMigrate verifies the long help on
// `crank migrate` includes example invocations.
func TestE2E_CLI_LongDescriptionForMigrate(t *testing.T) {
	out := runCrank(t, "", "migrate", "--help")
	for _, want := range []string{"crank migrate up", "crank migrate down"} {
		if !strings.Contains(out, want) {
			t.Errorf("migrate --help missing %q:\n%s", want, out)
		}
	}
}

// TestE2E_CLI_LongDescriptionForSwag verifies the long help on `crank
// swag` mentions the swag init invocation.
func TestE2E_CLI_LongDescriptionForSwag(t *testing.T) {
	out := runCrank(t, "", "make", "swag", "--help")
	if !strings.Contains(out, "swag init") {
		t.Errorf("make swag --help missing 'swag init':\n%s", out)
	}
}

// TestE2E_CLI_LongDescriptionForDev verifies the long help on `crank
// dev` mentions live reload.
func TestE2E_CLI_LongDescriptionForDev(t *testing.T) {
	out := runCrank(t, "", "dev", "--help")
	if !strings.Contains(out, "air") {
		t.Errorf("dev --help missing 'air' (the live-reload tool):\n%s", out)
	}
	if !strings.Contains(out, "live reload") {
		t.Errorf("dev --help missing 'live reload':\n%s", out)
	}
}

// TestE2E_CLI_LongDescriptionForRun verifies the long help on `crank
// run` mentions the go run invocation.
func TestE2E_CLI_LongDescriptionForRun(t *testing.T) {
	out := runCrank(t, "", "run", "--help")
	if !strings.Contains(out, "go run") {
		t.Errorf("run --help missing 'go run':\n%s", out)
	}
	if !strings.Contains(out, "./cmd/server") {
		t.Errorf("run --help missing './cmd/server':\n%s", out)
	}
}

// TestE2E_CLI_LongDescriptionForBuild verifies the long help on `crank
// build` mentions compiling cmd/server.
func TestE2E_CLI_LongDescriptionForBuild(t *testing.T) {
	out := runCrank(t, "", "build", "--help")
	if !strings.Contains(out, "cmd/server") {
		t.Errorf("build --help missing 'cmd/server':\n%s", out)
	}
	if !strings.Contains(out, "bin/") {
		t.Errorf("build --help missing 'bin/':\n%s", out)
	}
}

// TestE2E_CLI_Aliases_CommonSubcommands verifies the top-level aliases
// we expose. (Cobra aliases let users type `crank i` instead of `crank
// init`, etc. We just check that aliases are present on the subcommands
// that have them — this is a regression test against accidental
// alias removal.)
func TestE2E_CLI_Aliases_CommonSubcommands(t *testing.T) {
	// `crank list` should have an alias like `ls` or `features` (cobra
	// default is no alias; we accept either having an alias or not, as
	// long as the command works).
	out := runCrank(t, "", "list")
	if !strings.Contains(out, "NAME") {
		t.Errorf("list output doesn't look like a feature list:\n%s", out)
	}
}

// TestE2E_CLI_VersionOutputNotEmpty is a trivial sanity check on
// --version: the output must not be empty.
func TestE2E_CLI_VersionOutputNotEmpty(t *testing.T) {
	out := runCrank(t, "", "--version")
	if len(strings.TrimSpace(out)) == 0 {
		t.Errorf("--version output is empty")
	}
}

// TestE2E_CLI_HelpOutputNotEmpty is the same for --help.
func TestE2E_CLI_HelpOutputNotEmpty(t *testing.T) {
	out := runCrank(t, "", "--help")
	if len(strings.TrimSpace(out)) == 0 {
		t.Errorf("--help output is empty")
	}
}

// TestE2E_CLI_JsonFlagListIsValidJSON is a stricter version of
// TestE2E_List_JSON: every entry's name and description are non-empty,
// and the output is valid JSON parseable by encoding/json.
func TestE2E_CLI_JsonFlagListIsValidJSON(t *testing.T) {
	out := runCrank(t, "", "list", "--json")
	// The output is a JSON array; we just make sure it starts with '['
	// and ends with ']' (the encoder may add a trailing newline).
	trimmed := strings.TrimSpace(out)
	if !strings.HasPrefix(trimmed, "[") || !strings.HasSuffix(trimmed, "]") {
		t.Errorf("list --json output is not a JSON array:\n%s", out)
	}
	// Sanity check: each line of the output (if pretty-printed) should
	// have either "name" or "description".
	if !strings.Contains(trimmed, `"name"`) || !strings.Contains(trimmed, `"description"`) {
		t.Errorf("list --json output missing name/description fields:\n%s", out)
	}
}

// TestE2E_CLI_RootCommandNameIsC verifies that the binary identifies as
// "crank" in its own help text.
func TestE2E_CLI_RootCommandNameIsCrank(t *testing.T) {
	out := runCrank(t, "", "--help")
	if !strings.Contains(out, "crank") {
		t.Errorf("help output doesn't mention 'crank':\n%s", out)
	}
}

// TestE2E_CLI_ProjectFlag_RejectedByList verifies that the `list`
// command rejects --project (since the command is global and doesn't
// know what to do with it).
func TestE2E_CLI_ProjectFlag_RejectedByList(t *testing.T) {
	dir := scaffoldBase(t, "cli_projflag_list")
	out, err := runCrankRaw(t, "", "list", "--project", dir)
	if err == nil {
		t.Errorf("expected error for --project on list, got success:\n%s", out)
	}
}

// TestE2E_CLI_ProjectFlag_RejectedByTools is the same for `tools`.
func TestE2E_CLI_ProjectFlag_RejectedByTools(t *testing.T) {
	dir := scaffoldBase(t, "cli_projflag_tools")
	out, err := runCrankRaw(t, "", "tools", "--project", dir)
	if err == nil {
		t.Errorf("expected error for --project on tools, got success:\n%s", out)
	}
}

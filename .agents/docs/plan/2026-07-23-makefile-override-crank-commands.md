# Plan: Makefile Override of Native Crank Commands

**Status:** Exploration / Design Proposal  
**Date:** 2026-07-23  
**Author:** Zed (via Q&A with anurag925)

---

## 1. Background

The crank CLI has a **Makefile delegation** feature (`internal/bootstrap/commands/makedelegate.go`) that transparently forwards unknown subcommands to the target project's `Makefile`. For example, `crank clean` runs `make clean` if `clean` isn't a native crank command.

The question arose: **Can a Makefile override a built-in crank command** like `dev`, `build`, or `test`? Specifically, if a project's `Makefile` defines a `dev` target, should `crank dev` run crank's native `dev` wrapper or the project's `make dev`?

## 2. Current Behaviour

**Native commands always win.** The delegation logic (`TryMakeDelegation`) checks `isKnownCommand(root, candidate)` first. If the command name matches any registered crank subcommand (including tool commands like `dev`, `build`, `test`, etc.), it returns immediately — the Makefile is never consulted.

```go
if isKnownCommand(root, candidate) {
    return false, nil  // Makefile never considered
}
```

The code comment on lines 33–34 of `makedelegate.go` explicitly notes:

> *"Native crank commands always take precedence: the fallback is consulted only for names cobra does not already recognize. This means a Makefile can extend crank with project-specific targets, and (in the future) be used to override behavior for names crank does not ship natively."*

The "(in the future)" parenthetical indicates this override capability was anticipated but never implemented.

## 3. Why This Matters

- **Project-specific customisation.** A project may want to augment or replace crank's `dev` (air live-reload) with a custom hot-reload setup, or replace `test` with a custom test harness.
- **Monorepo flexibility.** Workspaces with complex build pipelines may need to intercept crank commands at the project level.
- **Gradual migration.** Teams adopting crank alongside existing Makefile-driven workflows may want the Makefile as the source of truth for some operations, with crank as a uniform entry point.

## 4. Options

### Option A: Opt-in Override via Manifest Flag

Add a field to `.crank.yaml` (e.g. `makefile_override: true`) that, when enabled, reverses the precedence — a Makefile target that shadows a crank command name wins over the native command.

```yaml
# .crank.yaml
crank_version: "1.0.0"
project_name: myapp
features: [base, auth, gorm]
makefile_override: true   # <-- new flag
```

**Pros:**
- Explicit, no silent breakage.
- Per-project opt-in.
- Crank retains "native first" by default.

**Cons:**
- Users must discover and set the flag.
- Another manifest field to maintain.

### Option B: Makefile Always Wins (Invert Precedence)

Change `TryMakeDelegation` to check the Makefile *before* the native command registry. If the Makefile defines the target, run it; otherwise fall through to cobra.

**Pros:**
- Maximum flexibility for project owners.
- No configuration needed.

**Cons:**
- Silent behavioural change — `crank dev` suddenly does something different.
- Breaks the contract that crank is the single source of truth for common dev tasks.
- Confusing for new contributors who expect crank's documented behaviour.

**Decision: Rejected.** Too risky and surprising.

### Option C: Per-command Makefile Passthrough

Add a `--passthrough` flag or environment variable for individual overrides:

```bash
crank dev --passthrough   # runs make dev instead of crank's dev
# or
CRANK_PASSTHROUGH=dev crank dev
```

**Pros:**
- Explicit at invocation time.
- No permanent configuration change.
- Useful for one-off debugging or testing.

**Cons:**
- Awkward for daily use (must remember the flag).
- Doesn't scale to CI or project-wide defaults.

### Option D: Crank-initiated Override via Makefile Comment Marker

Allow the project's Makefile to declare "I override this crank command" via a comment:

```makefile
# crank:override dev
dev:
	@echo "Custom dev runner"
```

`TryMakeDelegation` would scan the Makefile for `# crank:override <name>` markers and respect them during the known-command check.

**Pros:**
- Explicit, project-level declaration.
- Self-documenting — the override lives in the Makefile alongside the target.
- No new config fields.

**Cons:**
- Requires parsing Makefile comments (current code only extracts target names).
- Slightly more complex implementation.

### Option E: Project Skill Override (Preferred)

Use the existing agent skill mechanism: if the target project has a `.agents/skills/crank-project/SKILL.md` (which `crank update-skill` manages), it could declare overrides there. Crank reads the skill file at startup and skips native registration for any command listed as overridden.

Alternatively, a simpler variant: a file like `.crank-overrides.yaml` in the project root.

**Pros:**
- Decouples the override declaration from the Makefile.
- Extensible — could later support pre/post hooks, not just full replacement.

**Cons:**
- Another file to manage.
- Over-engineered for the current need.

## 5. Recommendation

**Adopt Option A (opt-in manifest flag).** It's the safest, most predictable approach:

1. Add `makefile_override bool` to the manifest struct.
2. In `TryMakeDelegation`, when `makefile_override` is `true`, consult the Makefile *before* checking `isKnownCommand`. If the Makefile defines the target, delegate to `make`; otherwise fall through to the native command.
3. Default to `false` — existing projects continue unchanged.
4. Document the flag in `crank init --help` and the `crank update` output.

### Implementation Sketch

**`internal/bootstrap/manifest.go`** — add field:
```go
type Manifest struct {
    CrankVersion     string   `yaml:"crank_version"`
    ProjectName      string   `yaml:"project_name"`
    ModulePath       string   `yaml:"module_path"`
    Features         []string `yaml:"features,omitempty"`
    MakefileOverride bool     `yaml:"makefile_override,omitempty"` // new
}
```

**`internal/bootstrap/commands/makedelegate.go`** — modify `TryMakeDelegation`:
```go
func TryMakeDelegation(root *cobra.Command, args []string) (handled bool, err error) {
    // ... existing preamble ...

    candidate := args[0]
    // ... existing flag check ...

    // Resolve project directory for both the Makefile and the manifest.
    projectDir, _ := splitProjectFlag(args[1:])

    // Check if Makefile override mode is enabled.
    override := isMakefileOverrideEnabled(projectDir)

    if override {
        // In override mode, check the Makefile first.
        if matchesMakefileTarget(projectDir, candidate) {
            return true, runMakeTarget(projectDir, candidate, ...)
        }
        // Otherwise fall through to native command.
    }

    // Default behaviour: native commands win.
    if isKnownCommand(root, candidate) {
        return false, nil
    }

    // ... existing Makefile fallback ...
}
```

Helper functions `isMakefileOverrideEnabled` and `matchesMakefileTarget` would load the manifest and make target list, respectively.

## 6. Open Questions

1. **Should the flag also affect `crank init`?** If `--makefile-override` is passed to `crank init`, should it automatically skip generating tool wrappers for overridden commands? Probably not — that's a separate concern.

2. **What about built-in commands like `crank init`, `crank add`, `crank list`?** These should never be overridable — they operate on the project manifest itself and a Makefile shouldn't intercept them. The check should exclude "meta" commands.

3. **Should the flag be settable via env var too?** (`CRANK_MAKEFILE_OVERRIDE=1`) — useful for CI.

## 7. Out of Scope (for now)

- Pre/post hooks around crank commands.
- Partial override (e.g. override `test` but add crank's native flags).
- Makefile command wrapping with argument transformation.

---

## References

- `internal/bootstrap/commands/makedelegate.go` — current delegation logic
- `internal/bootstrap/manifest.go` — manifest I/O
- `cmd/crank/main.go` — entry point where delegation is wired before `root.Execute()`
- Original discussion: "is make file able to override a crank command" (2026-07-23)

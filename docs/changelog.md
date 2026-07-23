# Changelog

## [Unreleased]

### Bun ORM support removed (breaking)

Bun is no longer a supported ORM. **GORM is now the sole persistence adapter**
(it covers PostgreSQL via its postgres driver). This is a breaking change for
projects generated with the `bun` feature.

**What changed:**

- The `bun` feature and all its templates have been deleted. `crank list` no
  longer shows it and `crank add bun` / `crank init --features=base,bun` now
  error with an unknown-feature message.
- The `--use-bun` flag on `crank init` has been removed, along with the
  interactive ORM-choice prompt. `crank init` always adds GORM by default when
  no ORM is requested.
- ORM mutual-exclusion logic is gone (there is only one ORM now).
- `outbox` and `audit` now require `gorm` (previously `bun` or `gorm`); their
  Bun-backed adapter templates have been removed.
- The `migrate` and `seed` tools now require the `gorm` feature.
- The seed generator (`crank make seed`) emits GORM-only seed files; the
  Bun seed path and the `--orm` distinction are gone.
- The `crank make` scaffold generator no longer has a Bun code path — the GORM
  adapter is generated for any project with the `gorm` feature.

See [the migration guide](migration-v2.md) for step-by-step upgrade
instructions.

### Seed/swag/skill moved under `crank make` (breaking)

The `seed`, `swag`, and `update-skill` subcommands have been moved from top-level commands to subcommands of `crank make`.

**Migration:**
- `crank seed generate User` → `crank make seed User`
- `crank seed up/down` → `crank make seed up/down`
- `crank swag` → `crank make swag`
- `crank update-skill` → `crank make skill`

### Exported Domain Fields (Clean Cutover)

The scaffold templates have been updated to follow the **exported-domain-fields**
pattern ("Edge pattern"). This is a clean cutover — there is no opt-in flag or
schema version; all newly generated resources use the new layout.

**What changed:**

- **Domain aggregate fields are now exported** (`ID`, `Name`, `CreatedAt`, etc.)
  with GORM tags directly on the struct. Per-field getters (`ID()`, `Name()`, …)
  have been removed — callers use direct field access (`x.ID`, `x.Name`).
- **No more `*Row` DTO.** The GORM repository uses the aggregate as the GORM
  model directly. `toAggregate()`, `RowFromAggregate()`, and `TableName()` on
  the row DTO have been removed (the aggregate now carries `TableName()`).
- **UUID fields use `uuid.UUID` natively** in the domain aggregate (was `string`).
  The HTTP DTO stays `string`; conversion happens in the handler layer.
- **Bun/postgres adapter template removed.** GORM is now the single supported
  persistence adapter. See the Bun-removal section above and the
  [migration guide](migration-v2.md).
- **Typed IDs (`{{Pascal}}ID`) dropped.** Raw `uuid.UUID` is used everywhere.
  The `domain_id_test.go.tmpl` has been deleted.
- **`New{{Pascal}}ID()` removed** from command handlers. Callers use
  `uuid.Parse(cmd.ID)` instead.

**Migration note:** Existing generated projects are unaffected — this change
only applies to resources scaffolded by newer versions of `crank`. To migrate
an existing resource to the new pattern, manually update the aggregate fields
to exported names, add GORM tags, move `TableName()`, and delete the row DTO.

For upgrade steps see the [migration guide](migration-v2.md).

package e2e

// Tests for the `outbox` feature. The outbox requires bun (it uses a
// database transaction to make the aggregate write and the outbox row
// commit atomically). These tests cover the requirement check, the
// multi-feature init path, and the migration that the feature contributes.

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestE2E_Outbox_RequiresBun confirms `crank add outbox` refuses to
// run on a base-only project.
func TestE2E_Outbox_RequiresBun(t *testing.T) {
	dir := scaffoldBase(t, "outbox_no_pg")
	out, err := runCrankRaw(t, "", "add", "outbox", "--project", dir)
	if err == nil {
		t.Fatalf("expected add outbox to fail without bun\n%s", out)
	}
	if !strings.Contains(out, "requires a database ORM") {
		t.Errorf("error should call out outbox ORM requirement, got:\n%s", out)
	}
}

// TestE2E_Outbox_Init_DefaultsToGorm confirms that `crank init --features=base,outbox`
// auto-adds gorm as the default ORM, so outbox gets the ORM it needs.
func TestE2E_Outbox_Init_DefaultsToGorm(t *testing.T) {
	dir := t.TempDir()
	_, err := runCrankRaw(t, dir, "init", "outbox_init_gorm", "--features=base,outbox")
	if err != nil {
		t.Fatalf("expected init with outbox (no explicit ORM) to succeed with gorm auto-added:\n%s", err)
	}
	manifest := readFile(t, filepath.Join(dir, "outbox_init_gorm"), ".crank.yaml")
	if !strings.Contains(manifest, "- gorm") {
		t.Errorf("manifest should include gorm (auto-added default ORM):\n%s", manifest)
	}
	if !strings.Contains(manifest, "- outbox") {
		t.Errorf("manifest should include outbox:\n%s", manifest)
	}
}

// TestE2E_Outbox_InitWithBun runs the init happy path with
// base+bun+outbox and verifies the generated project compiles and
// wires the bun-backed UoW plus the worker.
func TestE2E_Outbox_InitWithBun(t *testing.T) {
	dir := scaffold(t, "outbox_init", []string{"base", "bun", "outbox"})

	// Outbox-specific files exist.
	assertExists(t, dir, "internal/domain/outbox/event.go")
	assertExists(t, dir, "internal/domain/outbox/repository.go")
	assertExists(t, dir, "internal/adapters/persistence/bun/outbox_repository.go")
	assertExists(t, dir, "internal/adapters/outbox/bun_uow.go")
	assertExists(t, dir, "internal/adapters/outbox/worker.go")

	// The migration adds the outbox_events table.
	upMigration := readFile(t, dir, "migrations/000002_add_outbox_events.up.sql")
	if !strings.Contains(upMigration, "CREATE TABLE IF NOT EXISTS outbox_events") {
		t.Errorf("outbox up migration should create the outbox_events table:\n%s", upMigration)
	}

	// The composition root wires the bun UoW and starts the worker.
	main := readFile(t, dir, "cmd/server/main.go")
	if !strings.Contains(main, "outboxadapter.NewBunUoW") {
		t.Errorf("main.go should construct the bun UoW:\n%s", main)
	}
	if !strings.Contains(main, "outboxWorker.Run") {
		t.Errorf("main.go should start the outbox worker goroutine:\n%s", main)
	}
	if !strings.Contains(main, "bun.NewOutboxRepository") {
		t.Errorf("main.go should construct the outbox repository:\n%s", main)
	}

	// OutboxConfig is in the config struct.
	cfgGo := readFile(t, dir, "internal/config/config.go")
	if !strings.Contains(cfgGo, "OutboxConfig") {
		t.Errorf("config.go should contain OutboxConfig")
	}
	if !strings.Contains(cfgGo, `outbox.poll_interval`) {
		t.Errorf("config.go should set outbox.poll_interval default")
	}

	// Manifest lists all three features.
	manifest := readFile(t, dir, ".crank.yaml")
	for _, f := range []string{"base", "bun", "outbox"} {
		if !strings.Contains(manifest, "- "+f) {
			t.Errorf("manifest missing feature %q", f)
		}
	}

	// Project must compile and vet cleanly.
	compileProject(t, dir)
}

// TestE2E_Outbox_AddAfterBun confirms the add path: scaffold a
// base+bun project, then add outbox via the binary. The end state
// must match the init-with-all-three case.
//
// Note: this test exercises the add path's ability to write the new
// feature's files and update the manifest. It does NOT assert that
// main.go is re-rendered with outbox wiring — that is a known
// limitation of the add path (base's main.go has SkipIfExists) which
// callers work around by either initing with all features at once or
// running `crank make` to rewire the composition root.
func TestE2E_Outbox_AddAfterBun(t *testing.T) {
	dir := scaffold(t, "outbox_add", []string{"base", "bun"})

	runCrank(t, "", "add", "outbox", "--project", dir)

	// Outbox migration was added.
	assertExists(t, dir, "migrations/000002_add_outbox_events.up.sql")
	assertExists(t, dir, "migrations/000002_add_outbox_events.down.sql")

	// Outbox files were created.
	assertExists(t, dir, "internal/adapters/outbox/bun_uow.go")
	assertExists(t, dir, "internal/adapters/outbox/worker.go")
	assertExists(t, dir, "internal/adapters/persistence/bun/outbox_repository.go")

	// Manifest now lists outbox.
	manifest := readFile(t, dir, ".crank.yaml")
	if !strings.Contains(manifest, "- outbox") {
		t.Errorf("manifest should list outbox after add:\n%s", manifest)
	}

	// OutboxConfig block is injected into config.go.
	cfgGo := readFile(t, dir, "internal/config/config.go")
	if !strings.Contains(cfgGo, "OutboxConfig") {
		t.Errorf("config.go should contain OutboxConfig after outbox add")
	}
}

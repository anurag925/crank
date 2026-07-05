package scaffold_test

import (
	"strings"
	"testing"

	"github.com/anurag925/crank/internal/bootstrap/scaffold"
)

func TestGenerateWorkflowAndActivity(t *testing.T) {
	dir := newProject(t, []string{"temporal"})

	// Workflow generator: produces a workflow and registers it with the worker.
	wres, err := scaffold.Generate(scaffold.Options{
		ProjectDir: dir,
		Kind:       scaffold.KindWorkflow,
		Name:       "OrderFulfillment",
		Fields:     []string{"order_id:uuid"},
	})
	if err != nil {
		t.Fatalf("Generate workflow: %v", err)
	}
	if !exists(dir, "internal/adapters/temporal/workflow/order_fulfillment.go") {
		t.Error("expected internal/adapters/temporal/workflow/order_fulfillment.go")
	}
	assertParses(t, dir, "internal/adapters/temporal/workflow/order_fulfillment.go")
	if !wres.Wired {
		t.Errorf("expected workflow to be wired, hint=%q", wres.WireHint)
	}

	// Activity generator with companion test.
	ares, err := scaffold.Generate(scaffold.Options{
		ProjectDir: dir,
		Kind:       scaffold.KindActivity,
		Name:       "ChargeCard",
		Fields:     []string{"amount:float"},
		Tests:      true,
	})
	if err != nil {
		t.Fatalf("Generate activity: %v", err)
	}
	for _, rel := range []string{
		"internal/adapters/temporal/activity/charge_card.go",
		"internal/adapters/temporal/activity/charge_card_test.go",
	} {
		if !exists(dir, rel) {
			t.Errorf("expected %s", rel)
			continue
		}
		assertParses(t, dir, rel)
	}
	if !ares.Wired {
		t.Errorf("expected activity to be wired, hint=%q", ares.WireHint)
	}

	// The worker now registers workflows.
	worker := read(t, dir, "internal/adapters/temporal/worker.go")
	if !strings.Contains(worker, "w.RegisterWorkflow(workflow.OrderFulfillmentWorkflow)") {
		t.Errorf("worker.go missing Workflow registration:\n%s", worker)
	}
	assertParses(t, dir, "internal/adapters/temporal/worker.go")

	// Activities are registered in the Activities container.
	// The registration is unqualified because activities.go is inside package activity.
	acts := read(t, dir, "internal/adapters/temporal/activity/activities.go")
	if !strings.Contains(acts, "w.RegisterActivity(ChargeCardActivity)") {
		t.Errorf("activities.go missing Activity registration:\n%s", acts)
	}
	assertParses(t, dir, "internal/adapters/temporal/activity/activities.go")
}

func TestWorkflowGeneratorRequiresTemporalFeature(t *testing.T) {
	dir := newProject(t, []string{"bun"}) // temporal not enabled

	if _, err := scaffold.Generate(scaffold.Options{
		ProjectDir: dir,
		Kind:       scaffold.KindWorkflow,
		Name:       "Foo",
	}); err == nil {
		t.Fatal("expected an error generating a workflow without the temporal feature")
	}
	if _, err := scaffold.Generate(scaffold.Options{
		ProjectDir: dir,
		Kind:       scaffold.KindActivity,
		Name:       "Foo",
	}); err == nil {
		t.Fatal("expected an error generating an activity without the temporal feature")
	}
}

func TestTemporalWiringIsIdempotent(t *testing.T) {
	dir := newProject(t, []string{"temporal"})
	opts := scaffold.Options{ProjectDir: dir, Kind: scaffold.KindActivity, Name: "Notify"}

	if _, err := scaffold.Generate(opts); err != nil {
		t.Fatalf("first generate: %v", err)
	}
	forced := opts
	forced.Force = true
	if _, err := scaffold.Generate(forced); err != nil {
		t.Fatalf("regenerate: %v", err)
	}

	acts := read(t, dir, "internal/adapters/temporal/activity/activities.go")
	if n := strings.Count(acts, "w.RegisterActivity(NotifyActivity)"); n != 1 {
		t.Errorf("expected exactly one Notify registration in activities.go, got %d:\n%s", n, acts)
	}
}

package scaffold

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
)

// workerFile is the path (relative to the project root) of the Temporal worker
// aggregator that registers workflows and activities.
const workerFile = "internal/adapters/temporal/worker.go"

// Marker comments emitted by the temporal feature's worker template. New
// workflows and activities are registered immediately before these anchors.
const (
	markerWorkflowRegister = "// crank:workflow-register"
	markerActivityRegister = "// crank:activity-register"
)

// wireWorkflow registers a generated workflow with the project's Temporal
// worker (internal/adapters/temporal/worker.go).
func wireWorkflow(projectDir string, r Resource) (wireResult, error) {
	reg := fmt.Sprintf("w.RegisterWorkflow(workflow.%sWorkflow)", r.Pascal)
	hint := fmt.Sprintf(`could not auto-register the workflow in %s. Add this to registerWorkflows():

  %s`, workerFile, reg)
	return wireWorker(projectDir, markerWorkflowRegister, reg, hint)
}

// wireActivity registers a generated activity with the project's Temporal
// worker (internal/adapters/temporal/worker.go).
func wireActivity(projectDir string, r Resource) (wireResult, error) {
	reg := fmt.Sprintf("w.RegisterActivity(activity.%sActivity)", r.Pascal)
	hint := fmt.Sprintf(`could not auto-register the activity in %s. Add this to registerActivities():

  %s`, workerFile, reg)
	return wireWorker(projectDir, markerActivityRegister, reg, hint)
}

// wireWorker splices a registration line into worker.go immediately before the
// given marker. Like handler wiring it is best-effort and never corrupts the
// file: if the result does not format, the edit is discarded and a manual hint
// is returned. Re-registering the same line is a no-op (idempotent).
func wireWorker(projectDir, marker, regLine, hint string) (wireResult, error) {
	path := filepath.Join(projectDir, workerFile)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return wireResult{Hint: hint}, nil
	}
	if err != nil {
		return wireResult{}, fmt.Errorf("read %s: %w", workerFile, err)
	}

	content := string(data)
	if strings.Contains(content, regLine) {
		return wireResult{Wired: true}, nil
	}
	if !strings.Contains(content, marker) {
		return wireResult{Hint: hint}, nil
	}

	updated := strings.Replace(content, marker, regLine+"\n\t"+marker, 1)
	formatted, err := format.Source([]byte(updated))
	if err != nil {
		return wireResult{Hint: hint}, nil
	}
	if err := os.WriteFile(path, formatted, 0o644); err != nil {
		return wireResult{}, fmt.Errorf("write %s: %w", workerFile, err)
	}
	return wireResult{Wired: true}, nil
}

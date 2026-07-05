package scaffold

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
)

// workerFile is the path (relative to the project root) of the Temporal worker
// aggregator that registers workflows (and previously also activities).
const workerFile = "internal/adapters/temporal/worker.go"

// activitiesFile is the path of the Activities container that now owns the
// activity-registration marker.
const activitiesFile = "internal/adapters/temporal/activity/activities.go"

// Marker comments emitted by the temporal feature's templates. New workflows
// and activities are registered immediately before these anchors.
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
	return wireWorker(projectDir, workerFile, markerWorkflowRegister, reg, hint)
}

// wireActivity registers a generated activity with the project's Activities
// container (internal/adapters/temporal/activity/activities.go).
func wireActivity(projectDir string, r Resource) (wireResult, error) {
	reg := fmt.Sprintf("w.RegisterActivity(%sActivity)", r.Pascal)
	hint := fmt.Sprintf(`could not auto-register the activity in %s. Add this to Activities.Register():

  %s`, activitiesFile, reg)
	return wireWorker(projectDir, activitiesFile, markerActivityRegister, reg, hint)
}

// wireWorker splices a registration line into the given file immediately
// before the given marker. Like handler wiring it is best-effort and never
// corrupts the file: if the result does not format, the edit is discarded
// and a manual hint is returned. Re-registering the same line is a no-op
// (idempotent).
func wireWorker(projectDir, targetFile, marker, regLine, hint string) (wireResult, error) {
	path := filepath.Join(projectDir, targetFile)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		fmt.Printf("DEBUG wireWorker: file %s does not exist\n", path)
		return wireResult{Hint: hint}, nil
	}
	if err != nil {
		return wireResult{}, fmt.Errorf("read %s: %w", targetFile, err)
	}

	content := string(data)
	if strings.Contains(content, regLine) {
		return wireResult{Wired: true}, nil
	}
	if !strings.Contains(content, marker) {
		fmt.Printf("DEBUG wireWorker: marker %q not found in %s\ncontent:\n%s\n", marker, targetFile, content)
		return wireResult{Hint: hint}, nil
	}

	updated := strings.Replace(content, marker, regLine+"\n\t"+marker, 1)
	formatted, err := format.Source([]byte(updated))
	if err != nil {
		fmt.Printf("DEBUG wireWorker: format.Source failed for %s: %v\nupdated:\n%s\n", targetFile, err, updated)
		return wireResult{Hint: hint}, nil
	}
	if err := os.WriteFile(path, formatted, 0o644); err != nil {
		return wireResult{}, fmt.Errorf("write %s: %w", targetFile, err)
	}
	return wireResult{Wired: true}, nil
}

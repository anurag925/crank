package tidy

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/anurag925/crank/internal/bootstrap"
	"github.com/anurag925/crank/internal/utils"
)

func init() {
	bootstrap.GlobalToolRegistry.MustRegister(tool{})
}

type tool struct{}

func (tool) Name() string               { return "tidy" }
func (tool) BinaryName() string         { return "go" }
func (tool) Description() string        { return "Sync module dependencies (go mod tidy)" }
func (tool) RequiresFeatures() []string { return nil }

func (tool) LongDescription() string {
	return `tidy runs 'go mod tidy' inside the target project.

If --project is not specified, the current directory is used.

Examples:
  crank tidy --project ./myapp
  cd myapp && crank tidy                           (uses current directory)`
}

func (tool) InstallCmd() string          { return "" }
func (tool) Install() error              { return nil }
func (tool) AddFlags(cmd *cobra.Command) {}

func (tool) Prepare(projectDir string, cmd *cobra.Command, extraArgs []string) (*bootstrap.ToolInvocation, error) {
	goMod := filepath.Join(projectDir, "go.mod")
	if !utils.PathExists(goMod) {
		return nil, fmt.Errorf("no go.mod found in %s", projectDir)
	}

	argv := []string{"mod", "tidy"}
	argv = append(argv, extraArgs...)

	return &bootstrap.ToolInvocation{Args: argv, Dir: projectDir}, nil
}

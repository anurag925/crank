package build

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

func (tool) Name() string               { return "build" }
func (tool) BinaryName() string         { return "go" }
func (tool) Description() string        { return "Compile the project binary" }
func (tool) RequiresFeatures() []string { return nil }

func (tool) LongDescription() string {
	return `build compiles the project's cmd/server entry point into a binary
inside the bin/ directory.

If --project is not specified, the current directory is used.

Examples:
  crank build --project ./myapp
  cd myapp && crank build                          (uses current directory)`
}

func (tool) InstallCmd() string          { return "" }
func (tool) Install() error              { return nil }
func (tool) AddFlags(cmd *cobra.Command) {}

func (tool) Prepare(projectDir string, cmd *cobra.Command) (*bootstrap.ToolInvocation, error) {
	mainGo := filepath.Join(projectDir, "cmd", "server")
	if !utils.PathExists(mainGo) {
		return nil, fmt.Errorf("no cmd/server/ found in %s", projectDir)
	}

	argv := []string{"build", "-o", filepath.Join("bin", filepath.Base(projectDir)), "./cmd/server"}
	argv = append(argv, cmd.Flags().Args()...)

	return &bootstrap.ToolInvocation{Args: argv, Dir: projectDir}, nil
}

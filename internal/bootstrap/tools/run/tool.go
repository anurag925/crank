package run

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

func (tool) Name() string               { return "run" }
func (tool) BinaryName() string         { return "go" }
func (tool) Description() string        { return "Run the project server" }
func (tool) RequiresFeatures() []string { return nil }

func (tool) LongDescription() string {
	return `run starts the project server in the foreground via 'go run ./cmd/server'.

If --project is not specified, the current directory is used.

Examples:
  crank run --project ./myapp
  cd myapp && crank run                            (uses current directory)`
}

func (tool) InstallCmd() string          { return "" }
func (tool) Install() error              { return nil }
func (tool) AddFlags(cmd *cobra.Command) {}

func (tool) Prepare(projectDir string, cmd *cobra.Command) (*bootstrap.ToolInvocation, error) {
	mainGo := filepath.Join(projectDir, "cmd", "server")
	if !utils.PathExists(mainGo) {
		return nil, fmt.Errorf("no cmd/server/ found in %s", projectDir)
	}

	argv := []string{"run", "./cmd/server"}
	argv = append(argv, cmd.Flags().Args()...)

	return &bootstrap.ToolInvocation{Args: argv, Dir: projectDir, Stdin: true}, nil
}

package test

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/anurag925/rev/internal/bootstrap"
	"github.com/anurag925/rev/internal/utils"
)

func init() {
	bootstrap.GlobalToolRegistry.MustRegister(tool{})
}

type tool struct{}

func (tool) Name() string               { return "test" }
func (tool) BinaryName() string         { return "go" }
func (tool) Description() string        { return "Run all project tests" }
func (tool) RequiresFeatures() []string { return nil }

func (tool) LongDescription() string {
	return `test runs 'go test ./...' inside the target project.
Extra flags are forwarded to go test.

If --project is not specified, the current directory is used.

Examples:
  rev test --project ./myapp
  rev test -v -count=1 --project ./myapp
  cd myapp && rev test                           (uses current directory)`
}

func (tool) InstallCmd() string          { return "" }
func (tool) Install() error              { return nil }
func (tool) AddFlags(cmd *cobra.Command) {}

func (tool) Prepare(projectDir string, cmd *cobra.Command) (*bootstrap.ToolInvocation, error) {
	mainGo := filepath.Join(projectDir, "cmd", "server")
	if !utils.PathExists(mainGo) {
		return nil, fmt.Errorf("no cmd/server/ found in %s", projectDir)
	}

	argv := []string{"test", "./..."}
	argv = append(argv, cmd.Flags().Args()...)

	return &bootstrap.ToolInvocation{Args: argv, Dir: projectDir}, nil
}

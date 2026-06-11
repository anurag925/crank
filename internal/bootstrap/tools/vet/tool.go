package vet

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

func (tool) Name() string               { return "vet" }
func (tool) BinaryName() string         { return "go" }
func (tool) Description() string        { return "Run go vet on the project" }
func (tool) RequiresFeatures() []string { return nil }

func (tool) LongDescription() string {
	return `vet runs 'go vet ./...' inside the target project.

If --project is not specified, the current directory is used.

Examples:
  rev vet --project ./myapp
  cd myapp && rev vet                            (uses current directory)`
}

func (tool) InstallCmd() string          { return "" }
func (tool) Install() error              { return nil }
func (tool) AddFlags(cmd *cobra.Command) {}

func (tool) Prepare(projectDir string, cmd *cobra.Command) (*bootstrap.ToolInvocation, error) {
	mainGo := filepath.Join(projectDir, "cmd", "server")
	if !utils.PathExists(mainGo) {
		return nil, fmt.Errorf("no cmd/server/ found in %s", projectDir)
	}

	argv := []string{"vet", "./..."}
	argv = append(argv, cmd.Flags().Args()...)

	return &bootstrap.ToolInvocation{Args: argv, Dir: projectDir}, nil
}

package gofmt

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

func (tool) Name() string               { return "gofmt" }
func (tool) BinaryName() string         { return "gofmt" }
func (tool) Description() string        { return "Format Go source files" }
func (tool) RequiresFeatures() []string { return nil }

func (tool) LongDescription() string {
	return `gofmt runs 'gofmt -s -w .' inside the target project.

If --project is not specified, the current directory is used.

Examples:
  crank gofmt --project ./myapp
  crank gofmt -l --project ./myapp   (list files that differ)
  cd myapp && crank gofmt                       (uses current directory)`
}

func (tool) InstallCmd() string          { return "" }
func (tool) Install() error              { return nil }
func (tool) AddFlags(cmd *cobra.Command) {}

func (tool) Prepare(projectDir string, cmd *cobra.Command, extraArgs []string) (*bootstrap.ToolInvocation, error) {
	goMod := filepath.Join(projectDir, "go.mod")
	if !utils.PathExists(goMod) {
		return nil, fmt.Errorf("no go.mod found in %s", projectDir)
	}

	argv := []string{"-s", "-w", "."}
	argv = append(argv, extraArgs...)

	return &bootstrap.ToolInvocation{Args: argv, Dir: projectDir}, nil
}

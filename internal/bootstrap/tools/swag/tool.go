package swag

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/anurag925/crank/internal/bootstrap"
	"github.com/anurag925/crank/internal/bootstrap/tools"
	"github.com/anurag925/crank/internal/utils"
)

func init() {
	bootstrap.GlobalToolRegistry.MustRegister(tool{})
}

type tool struct{}

func (tool) Name() string               { return "swag" }
func (tool) BinaryName() string         { return "swag" }
func (tool) Description() string        { return "Generate Swagger/OpenAPI documentation" }
func (tool) RequiresFeatures() []string { return nil }

func (tool) LongDescription() string {
	return `swag generates Swagger 2.0 API documentation from Go annotations.
It runs 'swag init' against your project's cmd/server/main.go entry point,
producing docs/ with the generated OpenAPI spec.

If --project is not specified, the current directory is used.

Examples:
  crank swag --project ./myapp
  crank swag --parseDepth 2 --project ./myapp
  cd myapp && crank swag                          (uses current directory)`
}

func (tool) InstallCmd() string {
	return "go install github.com/swaggo/swag/cmd/swag@latest"
}

func (t tool) Install() error {
	return tools.InstallGoTool("github.com/swaggo/swag/cmd/swag@latest", t.BinaryName(), "")
}

func (tool) AddFlags(cmd *cobra.Command) {}

func (tool) Prepare(projectDir string, cmd *cobra.Command) (*bootstrap.ToolInvocation, error) {
	mainGo := filepath.Join(projectDir, "cmd", "server", "main.go")
	if !utils.PathExists(mainGo) {
		return nil, fmt.Errorf("no cmd/server/main.go found in %s", projectDir)
	}

	argv := []string{
		"init",
		"-g", "cmd/server/main.go",
		"-o", "docs",
		"--parseInternal",
		"--parseDependency",
	}
	argv = append(argv, cmd.Flags().Args()...)

	return &bootstrap.ToolInvocation{Args: argv, Dir: projectDir}, nil
}

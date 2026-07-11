package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/anurag925/crank/internal/bootstrap"
	"github.com/anurag925/crank/internal/bootstrap/tools"
	"github.com/anurag925/crank/internal/utils"
)

func init() {
	bootstrap.GlobalToolRegistry.MustRegister(&tool{})
}

type tool struct {
	databaseURL string
	steps       int
}

func (*tool) Name() string               { return "migrate" }
func (*tool) BinaryName() string         { return "migrate" }
func (*tool) Description() string        { return "Run database migrations via golang-migrate" }
func (*tool) RequiresFeatures() []string { return []string{"bun", "gorm"} }

func (*tool) LongDescription() string {
	return `migrate invokes the golang-migrate CLI inside the target project.
By default it applies all pending up migrations. The database URL is taken
from DATABASE_URL env var or the project's configs/config.yaml.

If --project is not specified, the current directory is used.

Examples:
  crank migrate up --project ./myapp
  crank migrate down --steps 1 --project ./myapp
  crank migrate --project ./myapp              (defaults to "up")
  cd myapp && crank migrate up                 (uses current directory)`
}

func (*tool) InstallCmd() string {
	return "go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
}

func (t *tool) Install() error {
	return tools.InstallGoTool("github.com/golang-migrate/migrate/v4/cmd/migrate@latest", t.BinaryName(), "postgres")
}

func (t *tool) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&t.databaseURL, "database-url", "", "override the database URL (defaults to DATABASE_URL or config)")
	cmd.Flags().IntVar(&t.steps, "steps", 0, "limit the number of migration steps (0 = all pending)")
}

func (t *tool) Prepare(projectDir string, cmd *cobra.Command, extraArgs []string) (*bootstrap.ToolInvocation, error) {
	if !utils.PathExists(filepath.Join(projectDir, "db/migrations")) {
		return nil, fmt.Errorf("no db/migrations/ directory found in %s", projectDir)
	}

	databaseURL := t.databaseURL
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		dsn, err := bootstrap.DSNFromConfig(projectDir)
		if err != nil {
			return nil, fmt.Errorf("could not determine database URL: %w", err)
		}
		databaseURL = dsn
	}

	direction := ""
	if len(extraArgs) > 0 {
		d := strings.ToLower(extraArgs[0])
		if d == "up" || d == "down" {
			direction = d
			extraArgs = extraArgs[1:]
		}
	}

	argv := []string{
		"-path", filepath.Join(projectDir, "db/migrations"),
		"-database", databaseURL,
	}
	if direction != "" {
		argv = append(argv, direction)
	}
	if t.steps > 0 {
		argv = append(argv, fmt.Sprintf("%d", t.steps))
	}
	argv = append(argv, extraArgs...)

	return &bootstrap.ToolInvocation{
		Args: argv,
		Dir:  projectDir,
	}, nil
}

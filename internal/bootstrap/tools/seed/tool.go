package seed

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

func (*tool) Name() string               { return "seed" }
func (*tool) BinaryName() string         { return "migrate" }
func (*tool) Description() string        { return "Run seed data migrations via golang-migrate" }
func (*tool) RequiresFeatures() []string { return []string{"bun", "gorm"} }

func (*tool) LongDescription() string {
	return `seed invokes the golang-migrate CLI pointing at db/seeds/ directory.
By default it applies all pending up seed files. The database URL is taken
from DATABASE_URL env var or the project's configs/config.yaml.

If --project is not specified, the current directory is used.

Examples:
  crank seed up --project ./myapp
  crank seed down --steps 1 --project ./myapp
  crank seed --project ./myapp              (defaults to "up")
  cd myapp && crank seed up                 (uses current directory)`
}

func (*tool) InstallCmd() string {
	return "go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
}

func (t *tool) Install() error {
	return tools.InstallGoTool("github.com/golang-migrate/migrate/v4/cmd/migrate@latest", t.BinaryName(), "postgres")
}

func (t *tool) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&t.databaseURL, "database-url", "", "override the database URL (defaults to DATABASE_URL or config)")
	cmd.Flags().IntVar(&t.steps, "steps", 0, "limit the number of seed steps (0 = all pending)")
}

func (t *tool) Prepare(projectDir string, cmd *cobra.Command, extraArgs []string) (*bootstrap.ToolInvocation, error) {
	seedsDir := filepath.Join(projectDir, "db/seeds")

	// Create the seeds directory if it doesn't exist (caller is free to
	// add timestamped SQL files later; an empty directory is a valid
	// golang-migrate source with zero migrations).
	if err := utils.EnsureDir(seedsDir); err != nil {
		return nil, fmt.Errorf("create db/seeds directory: %w", err)
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

	direction := "up"
	if len(extraArgs) > 0 {
		d := strings.ToLower(extraArgs[0])
		if d == "up" || d == "down" {
			direction = d
			extraArgs = extraArgs[1:]
		}
	}

	argv := []string{
		"-path", seedsDir,
		"-database", databaseURL,
		direction,
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

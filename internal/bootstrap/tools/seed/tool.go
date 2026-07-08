package seed

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/anurag925/crank/internal/bootstrap"
	"github.com/anurag925/crank/internal/bootstrap/seedgen"
	"github.com/anurag925/crank/internal/bootstrap/tools"
	"github.com/anurag925/crank/internal/utils"
)

func init() {
	bootstrap.GlobalToolRegistry.MustRegister(&tool{})
}

type tool struct {
	databaseURL string
	steps       int
	count       int
	force       bool
}

func (*tool) Name() string       { return "seed" }
func (*tool) BinaryName() string { return "migrate" }
func (*tool) Description() string {
	return "Run seed data migrations or generate seed files via golang-migrate"
}
func (*tool) RequiresFeatures() []string { return []string{"bun", "gorm"} }

func (*tool) LongDescription() string {
	return `seed manages seed data in a crank-generated project.

Subcommands:
  crank seed [up|down]            Apply or rollback seed SQL files using golang-migrate
  crank seed generate <model>     Generate a seed SQL file with fake data for a domain model
  crank seed generate             Generate an empty seed file for manual editing

If no subcommand is given, "up" is assumed.

Examples:
  crank seed up --project ./myapp
  crank seed down --steps 1 --project ./myapp
  crank seed generate User --project ./myapp
  crank seed generate --count 20 --project ./myapp
  crank seed --project ./myapp                    (defaults to "up")`
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
	cmd.Flags().IntVar(&t.count, "count", 10, "number of seed rows to generate (used with 'generate')")
	cmd.Flags().BoolVar(&t.force, "force", false, "overwrite existing seed file (used with 'generate')")
}

func (t *tool) Prepare(projectDir string, cmd *cobra.Command, extraArgs []string) (*bootstrap.ToolInvocation, error) {
	// Check if the first positional arg is "generate".
	if len(extraArgs) > 0 && strings.ToLower(extraArgs[0]) == "generate" {
		if err := t.handleGenerate(projectDir, extraArgs[1:]); err != nil {
			return nil, err
		}
		// Signal to executeTool that the work is done (no binary to run).
		return nil, nil
	}

	// Original migrate-wrapper path for up/down.
	seedsDir := filepath.Join(projectDir, "db/seeds")
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

// handleGenerate runs the seed file generator. It reads the struct from the
// domain layer, generates fake data, and writes a timestamped SQL file to
// db/seeds/.
func (t *tool) handleGenerate(projectDir string, args []string) error {
	modelName := ""
	if len(args) > 0 {
		modelName = args[0]
	}

	files, err := seedgen.GenerateSeed(seedgen.Options{
		ProjectDir: projectDir,
		ModelName:  modelName,
		Count:      t.count,
		Force:      t.force,
	})
	if err != nil {
		return err
	}

	for _, f := range files {
		if f.Skipped {
			fmt.Printf("    = db/seeds/%s (exists, skipped)\n", f.Path)
		} else {
			fmt.Printf("    + db/seeds/%s\n", f.Path)
		}
	}

	if modelName != "" {
		fmt.Printf("✔ Generated seed data for %s (%d rows)\n", modelName, t.count)
	} else {
		fmt.Println("✔ Generated empty seed file")
	}

	return nil
}

package seed

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/anurag925/crank/internal/bootstrap"
	"github.com/anurag925/crank/internal/bootstrap/seedgen"
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
func (*tool) BinaryName() string { return "go" }
func (*tool) Description() string {
	return "Generate and run Go-based seed data for your project"
}
func (*tool) RequiresFeatures() []string { return []string{"gorm"} }

func (*tool) LongDescription() string {
	return `seed manages seed data in a crank-generated project using Go-based seed files.

Subcommands:
  crank seed [up|down]            Apply or rollback seed data (go run db/seeds/main.go)
  crank seed generate <model>     Generate a Go seed file with fake data for a domain model
  crank seed generate             Generate scaffolding files (main.go + seeder.go)

If no subcommand is given, "up" is assumed.

Generated files:
  db/seeds/main.go                Entry point — connects to DB, runs seeder
  db/seeds/gorm/seeder.go         Orchestrator — registers all seed up/down functions
  db/seeds/gorm/seed_<table>.go   Per-model seed file with SeedXxxUp / SeedXxxDown

Examples:
  crank seed up --project ./myapp
  crank seed down --project ./myapp
  crank seed generate User --project ./myapp
  crank seed generate --count 5 --project ./myapp
  crank seed --project ./myapp                    (defaults to "up")`
}

func (*tool) InstallCmd() string {
	return "go is required to run seed operations"
}

func (t *tool) Install() error {
	return nil
}

func (t *tool) AddFlags(cmd *cobra.Command) {
	cmd.Flags().IntVar(&t.count, "count", 10, "number of seed rows to generate (used with 'generate')")
	cmd.Flags().BoolVar(&t.force, "force", false, "overwrite existing seed file (used with 'generate')")
}

func (t *tool) Prepare(projectDir string, cmd *cobra.Command, extraArgs []string) (*bootstrap.ToolInvocation, error) {
	if len(extraArgs) > 0 && strings.ToLower(extraArgs[0]) == "generate" {
		if err := t.handleGenerate(projectDir, extraArgs[1:]); err != nil {
			return nil, err
		}
		return nil, nil
	}

	return t.handleUpDown(projectDir, extraArgs)
}

func (t *tool) handleUpDown(projectDir string, extraArgs []string) (*bootstrap.ToolInvocation, error) {
	mainPath := filepath.Join(projectDir, "db", "seeds", "main.go")
	if !utils.PathExists(mainPath) {
		return nil, fmt.Errorf("db/seeds/main.go not found — run 'crank seed generate' first to scaffold seed files")
	}

	direction := "up"
	if len(extraArgs) > 0 {
		d := strings.ToLower(extraArgs[0])
		if d == "up" || d == "down" {
			direction = d
		}
	}

	fmt.Printf("→ go run db/seeds/main.go -dir %s\n", direction)
	ecmd := exec.Command("go", "run", "./db/seeds/main.go", "-dir", direction)
	ecmd.Dir = projectDir
	ecmd.Stdout = os.Stdout
	ecmd.Stderr = os.Stderr
	if err := ecmd.Run(); err != nil {
		return nil, fmt.Errorf("seed %s: %w", direction, err)
	}
	return nil, nil
}

func (t *tool) handleGenerate(projectDir string, args []string) error {
	proj, err := bootstrap.LoadProjectInfo(projectDir)
	if err != nil {
		return fmt.Errorf("load project info: %w", err)
	}

	modelName := ""
	if len(args) > 0 {
		modelName = args[0]
	}

	files, err := seedgen.GenerateSeed(seedgen.Options{
		ProjectDir: projectDir,
		ModelName:  modelName,
		Count:      t.count,
		Force:      t.force,
		ModulePath: proj.ModulePath,
	})
	if err != nil {
		return err
	}

	for _, f := range files {
		if f.Skipped {
			fmt.Printf("    = %s (exists, skipped)\n", f.Path)
		} else {
			fmt.Printf("    + %s\n", f.Path)
		}
	}

	if modelName != "" {
		fmt.Printf("✔ Generated seed data for %s (%d rows)\n", modelName, t.count)
	} else {
		fmt.Println("✔ Generated seed scaffolding (main.go + seeder.go)")
	}

	return nil
}

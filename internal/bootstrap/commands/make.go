package commands

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"

	"github.com/anurag925/crank/internal/bootstrap"
	"github.com/anurag925/crank/internal/bootstrap/scaffold"
	"github.com/anurag925/crank/internal/bootstrap/seedgen"
	"github.com/anurag925/crank/internal/bootstrap/tools"
	"github.com/anurag925/crank/internal/utils"
)

const makeLongDesc = `Generate boilerplate code: model, repository, service, handler, scaffold, workflow, activity, migration, seed, swag, skill.

Kinds:
  model       Domain struct (+ migration if gorm)
  repository  Data-access layer (GORM-backed or in-memory)
  service     CRUD service layer
  handler     HTTP handler (Echo router)
  scaffold    Full stack (model + repo + handler + wiring)
  workflow    Temporal workflow
  activity    Temporal activity
  migration   Blank SQL migration pair
  seed        Generate seed data files or run seed up/down
  swag        Generate Swagger/OpenAPI documentation
  skill       Regenerate the crank-project agent SKILL.md

Fields: optional "name:type" pairs. Types: string, text, int, int64, float, float64, bool, time, uuid, email

Examples:
  crank make model Order
  crank make model Order customer:string total:float
  crank make handler Product title:string price:float
  crank make handler Product --only
  crank make scaffold Invoice number:string amount:float
  crank make scaffold Invoice number:string --tests
  crank make scaffold Product --seed
  crank make workflow OrderFulfillment order_id:uuid
  crank make activity ChargeCard amount:float --tests
  crank make repository Ticket
  crank make migration add_index_to_orders
  crank make seed User
  crank make seed User --count 20 --force
  crank make seed
  crank make swag
  crank make skill`

// NewMakeCmd returns the `make` cobra command which generates scaffolding inside
// an existing project (models, repositories, services, handlers, migrations, seeds, etc).
func NewMakeCmd(featReg *bootstrap.Registry, toolReg *bootstrap.ToolRegistry) *cobra.Command {
	var (
		projectDir    string
		only          bool
		force         bool
		skipMigration bool
		tests         bool
		seed          bool
		count         int
	)

	cmd := &cobra.Command{
		Use:   "make <kind> <name> [field:type ...]",
		Short: "Generate models, handlers, migrations, seeds, and more",
		Long:  makeLongDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				cmd.Help()
				return nil
			}
			kind := args[0]
			name := ""
			if len(args) > 1 {
				name = args[1]
			}
			var fields []string
			if len(args) > 2 {
				fields = args[2:]
			}

			validKinds := []string{"migration", "seed", "swag", "skill"}
			validKinds = append(validKinds, scaffold.Kinds()...)

			switch kind {
			case "migration":
				if name == "" {
					return fmt.Errorf("migration name is required\n\nUsage: crank make %s <name>\n\nExample: crank make migration create_users_table", kind)
				}
				return makeMigration(projectDir, name)

			case "seed":
				return handleSeed(projectDir, args[1:], count, force)

			case "swag":
				return handleSwag(projectDir, args[1:])

			case "skill":
				return handleSkill(featReg, projectDir)

			case scaffold.KindModel,
				scaffold.KindRepository,
				scaffold.KindService,
				scaffold.KindHandler,
				scaffold.KindScaffold,
				scaffold.KindWorkflow,
				scaffold.KindActivity:
				if name == "" {
					return fmt.Errorf("%s name is required\n\nUsage: crank make %s <name> [field:type ...]\n\nExample: crank make %s Order customer:string total:float", kind, kind, kind)
				}
				if err := runScaffold(scaffold.Options{
					ProjectDir:    projectDir,
					Kind:          kind,
					Name:          name,
					Fields:        fields,
					Only:          only,
					Force:         force,
					SkipMigration: skipMigration,
					Tests:         tests,
				}); err != nil {
					return err
				}
				if seed {
					return generateSeed(projectDir, name, count, force)
				}
				return nil

			default:
				return fmt.Errorf("unknown kind %q\n\nSupported kinds: %s\n\nTip: Run 'crank make --help' to see all available kinds and examples.", kind, strings.Join(validKinds, ", "))
			}
		},
	}

	cmd.Flags().StringVar(&projectDir, "project", ".", "path to the crank project directory")
	cmd.Flags().BoolVar(&only, "only", false, "generate only the requested kind, skipping dependencies (model, repo/service)")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing files (used with scaffold and seed)")
	cmd.Flags().BoolVar(&skipMigration, "skip-migration", false, "do not generate a table migration even when the gorm feature is enabled")
	cmd.Flags().BoolVar(&tests, "tests", false, "generate _test.go files alongside each generated layer")
	cmd.Flags().BoolVar(&seed, "seed", false, "also generate a seed file (used with scaffold/model/handler)")
	cmd.Flags().IntVar(&count, "count", 10, "number of seed rows to generate (used with 'make seed')")
	return cmd
}

// runScaffold executes a code generator and prints a human-friendly summary.
func runScaffold(opts scaffold.Options) error {
	result, err := scaffold.Generate(opts)
	if err != nil {
		return err
	}

	for _, f := range result.Created {
		fmt.Println("    + " + f)
	}
	for _, f := range result.Skipped {
		fmt.Println("    = " + f + " (exists, skipped)")
	}

	if result.Wired {
		fmt.Printf("✔ Registered %sHandler in %s\n", result.Resource.Pascal, "internal/adapters/http/web/routes.go")
	}
	if result.WireHint != "" {
		fmt.Println("⚠ " + result.WireHint)
	}

	fmt.Printf("✔ Generated %s (%d file(s))\n", result.Resource.Pascal, len(result.Created))
	return nil
}

func makeMigration(projectDir, name string) error {
	name = sanitizeName(name)
	dir := filepath.Join(projectDir, "db/migrations")
	if err := utils.EnsureDir(dir); err != nil {
		return err
	}

	stamp := scaffold.NextMigrationVersion(dir)
	up := filepath.Join(dir, fmt.Sprintf("%s_%s.up.sql", stamp, name))
	down := filepath.Join(dir, fmt.Sprintf("%s_%s.down.sql", stamp, name))

	if err := utils.WriteFile(up, "-- +migrate Up\n"); err != nil {
		return err
	}
	if err := utils.WriteFile(down, "-- +migrate Down\n"); err != nil {
		return err
	}

	fmt.Printf("✔ Created %s\n✔ Created %s\n", up, down)
	return nil
}

func sanitizeName(s string) string {
	var b strings.Builder
	prevUnderscore := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevUnderscore = false
		case r == '-' || r == '_' || r == ' ':
			if !prevUnderscore && b.Len() > 0 {
				b.WriteRune('_')
				prevUnderscore = true
			}
		}
	}
	out := strings.TrimRight(b.String(), "_")
	if out == "" {
		fmt.Fprintln(os.Stderr, "warning: migration name produced an empty slug, defaulting to `change`")
		return "change"
	}
	return out
}

// handleSeed handles `crank make seed [<model>|up|down]`.
func handleSeed(projectDir string, args []string, count int, force bool) error {
	if len(args) == 0 {
		// No args — scaffold seed infrastructure (main.go + seeder.go) only.
		return generateSeed(projectDir, "", count, force)
	}

	arg := args[0]
	if arg == "up" || arg == "down" {
		return runSeed(projectDir, arg)
	}

	// Treat as a model name — generate seed data for it.
	return generateSeed(projectDir, arg, count, force)
}

// runSeed executes `go run ./db/seeds/main.go -dir <direction>`.
func runSeed(projectDir, direction string) error {
	mainPath := filepath.Join(projectDir, "db", "seeds", "main.go")
	if !utils.PathExists(mainPath) {
		return fmt.Errorf("db/seeds/main.go not found — run 'crank make seed' first to scaffold seed files")
	}

	fmt.Printf("→ go run db/seeds/main.go -dir %s\n", direction)
	ecmd := exec.Command("go", "run", "./db/seeds/main.go", "-dir", direction)
	ecmd.Dir = projectDir
	ecmd.Stdout = os.Stdout
	ecmd.Stderr = os.Stderr
	if err := ecmd.Run(); err != nil {
		return fmt.Errorf("seed %s: %w", direction, err)
	}
	return nil
}

// generateSeed generates seed data files for a domain model.
// When modelName is empty, only scaffolding (main.go + seeder.go) is generated.
func generateSeed(projectDir, modelName string, count int, force bool) error {
	proj, err := bootstrap.LoadProjectInfo(projectDir)
	if err != nil {
		return fmt.Errorf("load project info: %w", err)
	}

	files, err := seedgen.GenerateSeed(seedgen.Options{
		ProjectDir: projectDir,
		ModelName:  modelName,
		Count:      count,
		Force:      force,
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
		fmt.Printf("✔ Generated seed data for %s (%d rows)\n", modelName, count)
	} else {
		fmt.Println("✔ Generated seed scaffolding (main.go + seeder.go)")
	}

	return nil
}

// handleSwag handles `crank make swag [args...]`.
func handleSwag(projectDir string, extraArgs []string) error {
	mainGo := filepath.Join(projectDir, "cmd", "server", "main.go")
	if !utils.PathExists(mainGo) {
		return fmt.Errorf("no cmd/server/main.go found in %s", projectDir)
	}

	// Find or auto-install the swag binary.
	bin, err := utils.FindBinary("swag", "go install github.com/swaggo/swag/cmd/swag@latest")
	if err != nil {
		fmt.Printf("⚠ swag not found. Attempting to install...\n")
		if installErr := tools.InstallGoTool("github.com/swaggo/swag/cmd/swag@latest", "swag", ""); installErr != nil {
			return fmt.Errorf("auto-install failed: %w\n  Manual install: go install github.com/swaggo/swag/cmd/swag@latest", installErr)
		}
		bin, err = utils.FindBinary("swag", "go install github.com/swaggo/swag/cmd/swag@latest")
		if err != nil {
			return err
		}
	}

	argv := []string{
		"init",
		"-g", "cmd/server/main.go",
		"-o", "docs",
		"--parseInternal",
		"--parseDependency",
	}
	argv = append(argv, extraArgs...)

	fmt.Printf("→ swag %s\n", strings.Join(argv, " "))
	return utils.RunExternal(&utils.ExecConfig{
		Binary: bin,
		Args:   argv,
		Dir:    projectDir,
	})
}

// handleSkill handles `crank make skill` — regenerates the crank-project agent SKILL.md.
func handleSkill(reg *bootstrap.Registry, projectDir string) error {
	info, err := bootstrap.LoadProjectInfo(projectDir)
	if err != nil {
		return fmt.Errorf("load project info: %w", err)
	}

	base, ok := reg.Get("base")
	if !ok {
		return fmt.Errorf("base feature not found in registry")
	}

	const skillTemplatePath = "templates/agents_skills_crank_project_SKILL.md.tmpl"

	tmplData, err := fs.ReadFile(base.Templates(), skillTemplatePath)
	if err != nil {
		return fmt.Errorf("read skill template: %w", err)
	}

	tmpl, err := template.New("SKILL.md").Option("missingkey=error").Parse(string(tmplData))
	if err != nil {
		return fmt.Errorf("parse skill template: %w", err)
	}

	ctx := bootstrap.NewContext(info.ProjectName, info.ModulePath, info.Features)

	var sb strings.Builder
	if err := tmpl.Execute(&sb, ctx); err != nil {
		return fmt.Errorf("execute skill template: %w", err)
	}

	const skillOutputRelPath = ".agents/skills/crank-project/SKILL.md"
	dest := filepath.Join(projectDir, skillOutputRelPath)
	if err := utils.EnsureDir(filepath.Dir(dest)); err != nil {
		return fmt.Errorf("create skill dir: %w", err)
	}
	if err := os.WriteFile(dest, []byte(sb.String()), 0o644); err != nil {
		return fmt.Errorf("write skill file: %w", err)
	}

	fmt.Printf("✔ Updated %s\n", skillOutputRelPath)
	return nil
}

package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/anurag925/crank/internal/bootstrap/scaffold"
	"github.com/anurag925/crank/internal/utils"
)

const makeLongDesc = `Generate boilerplate code: model, repository, service, handler, scaffold, workflow, activity, migration.

Kinds:
  model       Domain struct (+ migration if gorm)
  repository  Data-access layer (GORM-backed or in-memory)
  service     CRUD service layer
  handler     HTTP handler (Echo router)
  scaffold    Full stack (model + repo + handler + wiring)
  workflow    Temporal workflow
  activity    Temporal activity
  migration   Blank SQL migration pair

Fields: optional "name:type" pairs. Types: string, text, int, int64, float, float64, bool, time, uuid, email

Examples:
  crank make model Order
  crank make model Order customer:string total:float
  crank make handler Product title:string price:float
  crank make handler Product --only
  crank make scaffold Invoice number:string amount:float
  crank make scaffold Invoice number:string --tests
  crank make workflow OrderFulfillment order_id:uuid
  crank make activity ChargeCard amount:float --tests
  crank make repository Ticket
  crank make migration add_index_to_orders`

// NewMakeCmd returns the `make` cobra command which generates scaffolding inside
// an existing project (models, repositories, services, handlers and migrations).
func NewMakeCmd() *cobra.Command {
	var (
		projectDir    string
		only          bool
		force         bool
		skipMigration bool
		tests         bool
	)

	cmd := &cobra.Command{
		Use:   "make <kind> <name> [field:type ...]",
		Short: "Generate models, handlers, migrations, and more",
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

			validKinds := []string{"migration"}
			validKinds = append(validKinds, scaffold.Kinds()...)

			switch kind {
			case "migration":
				if name == "" {
					return fmt.Errorf("migration name is required\n\nUsage: crank make %s <name>\n\nExample: crank make migration create_users_table", kind)
				}
				return makeMigration(projectDir, name)
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
				return runScaffold(scaffold.Options{
					ProjectDir:    projectDir,
					Kind:          kind,
					Name:          name,
					Fields:        fields,
					Only:          only,
					Force:         force,
					SkipMigration: skipMigration,
					Tests:         tests,
				})
			default:
				return fmt.Errorf("unknown kind %q\n\nSupported kinds: %s\n\nTip: Run 'crank make --help' to see all available kinds and examples.", kind, strings.Join(validKinds, ", "))
			}
		},
	}

	cmd.Flags().StringVar(&projectDir, "project", ".", "path to the crank project directory")
	cmd.Flags().BoolVar(&only, "only", false, "generate only the requested kind, skipping dependencies (model, repo/service)")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite the target file if it already exists")
	cmd.Flags().BoolVar(&skipMigration, "skip-migration", false, "do not generate a table migration even when the gorm feature is enabled")
	cmd.Flags().BoolVar(&tests, "tests", false, "generate _test.go files alongside each generated layer")
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

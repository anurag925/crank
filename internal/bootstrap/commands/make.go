package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/anurag925/crank/internal/bootstrap/scaffold"
	"github.com/anurag925/crank/internal/utils"
)

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
		Short: "Generate project scaffolding (models, repositories, services, handlers, migrations)",
		Long: `make generates boilerplate inside a project created by ` + "`init`" + `, in the
spirit of Rails generators and Laravel's artisan make commands.

Kinds:
  model       A domain model (plus a create-table migration when postgres is enabled).
  repository  A repository (Bun-backed with postgres, in-memory otherwise) + its model.
  service     An in-memory service layer + its model.
  handler     An HTTP CRUD handler + its model + repository/service, auto-wired into
              the Echo router (and a migration when postgres is enabled).
  scaffold    The full stack: model + repository/service + handler + migration + wiring.
  migration   A blank SQL migration pair in the migrations/ directory.

Fields are optional "name:type" pairs that populate the model, validation tags
and migration columns. Supported types: string, text, int, int64, float, bool,
time, uuid, email (defaulting to string).

Pass --tests to also generate _test.go files for each generated layer.

Examples:
  crank make model Order
  crank make model Order customer:string total:float paid:bool
  crank make handler Product title:string price:float        # handler + model + repo/service + wiring
  crank make handler Product --only                          # just the handler
  crank make scaffold Invoice number:string amount:float     # the whole stack
  crank make scaffold Invoice number:string --tests          # the whole stack + tests
  crank make repository Ticket --project ./myapp
  crank make migration add_index_to_orders`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind := args[0]
			name := ""
			if len(args) > 1 {
				name = args[1]
			}
			fields := args[2:]

			switch kind {
			case "migration":
				return makeMigration(projectDir, name)
			case scaffold.KindModel,
				scaffold.KindRepository,
				scaffold.KindService,
				scaffold.KindHandler,
				scaffold.KindScaffold:
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
				return fmt.Errorf("unknown kind %q (supported: %s, migration)", kind, strings.Join(scaffold.Kinds(), ", "))
			}
		},
	}

	cmd.Flags().StringVar(&projectDir, "project", ".", "path to the project directory")
	cmd.Flags().BoolVar(&only, "only", false, "generate only the requested artifact, skipping dependencies (handler/repository/service)")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite the target file if it already exists")
	cmd.Flags().BoolVar(&skipMigration, "skip-migration", false, "do not generate a table migration even when postgres is enabled")
	cmd.Flags().BoolVar(&tests, "tests", false, "also generate _test.go files for each generated model/repository/service/handler")
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
		fmt.Printf("✔ Registered %sHandler in %s\n", result.Resource.Pascal, "internal/handler/handler.go")
	}
	if result.WireHint != "" {
		fmt.Println("⚠ " + result.WireHint)
	}

	fmt.Printf("✔ Generated %s (%d file(s))\n", result.Resource.Pascal, len(result.Created))
	return nil
}

func makeMigration(projectDir, name string) error {
	if name == "" {
		return fmt.Errorf("migration name is required (e.g. `make migration create_users_table`)")
	}
	name = sanitizeName(name)
	dir := filepath.Join(projectDir, "migrations")
	if err := utils.EnsureDir(dir); err != nil {
		return err
	}

	stamp := time.Now().UTC().Format("20060102150405")
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

package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/anurag925/crank/internal/utils"
)

// NewMakeCmd returns the `make` cobra command which generates scaffolding inside an
// existing project.
func NewMakeCmd() *cobra.Command {
	var projectDir string

	cmd := &cobra.Command{
		Use:   "make <kind> <name>",
		Short: "Generate project scaffolding (e.g. migrations)",
		Long: `make generates boilerplate inside a project created by ` + "`init`" + `.

Currently supported kinds:
  migration  Create a new SQL migration pair in the migrations/ directory.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind := args[0]
			name := ""
			if len(args) > 1 {
				name = args[1]
			}

			switch kind {
			case "migration":
				return makeMigration(projectDir, name)
			default:
				return fmt.Errorf("unknown kind %q (supported: migration)", kind)
			}
		},
	}

	cmd.Flags().StringVar(&projectDir, "project", ".", "path to the project directory")
	return cmd
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

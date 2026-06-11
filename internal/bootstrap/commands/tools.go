package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/anurag925/crank/internal/bootstrap"
	"github.com/anurag925/crank/internal/utils"
)

// NewToolCmd creates a cobra command for a registered tool.
// The --project flag (defaulting to "."), tool flag registration, binary lookup,
// install prompt, and execution are all handled generically.
func NewToolCmd(reg *bootstrap.ToolRegistry, name string) (*cobra.Command, error) {
	t, ok := reg.Get(name)
	if !ok {
		return nil, fmt.Errorf("tool %q not found in registry", name)
	}

	var projectDir string

	cmd := &cobra.Command{
		Use:                t.Name(),
		Short:              t.Description(),
		Long:               t.LongDescription(),
		DisableFlagParsing: false,
		SilenceUsage:       true,
		SilenceErrors:      true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeTool(t, projectDir, cmd)
		},
	}

	cmd.Flags().StringVar(&projectDir, "project", ".", "path to the project directory (defaults to current directory)")

	// Let the tool register its own flags.
	t.AddFlags(cmd)

	return cmd, nil
}

// executeTool runs the full lifecycle: validate requirements → prepare → find binary → run.
func executeTool(t bootstrap.Tool, projectDir string, cmd *cobra.Command) error {
	// 1. Validate feature requirements against the project manifest.
	if err := bootstrap.ValidateToolRequirements(projectDir, t); err != nil {
		return err
	}

	// 2. Let the tool build its invocation.
	inv, err := t.Prepare(projectDir, cmd)
	if err != nil {
		return err
	}

	// 3. Resolve the binary.
	bin := inv.Binary
	if bin == "" {
		b, err := utils.FindBinary(t.BinaryName(), t.InstallCmd())
		if err != nil {
			// Prompt for auto-install.
			if t.InstallCmd() != "" {
				fmt.Printf("⚠ %s not found. Attempting to install...\n", t.BinaryName())
				if installErr := t.Install(); installErr != nil {
					return fmt.Errorf("auto-install failed: %w\n  Manual install: %s", installErr, t.InstallCmd())
				}
				// Retry lookup after install.
				b, err = utils.FindBinary(t.BinaryName(), t.InstallCmd())
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}
		bin = b
	}
	inv.Binary = bin

	// 4. Execute.
	fmt.Printf("→ %s %s\n", t.BinaryName(), strings.Join(inv.Args, " "))
	return utils.RunExternal(&utils.ExecConfig{
		Binary: inv.Binary,
		Args:   inv.Args,
		Dir:    inv.Dir,
		Env:    inv.Env,
		Stdin:  inv.Stdin,
	})
}

// NewToolsListCmd returns a `tools` command that lists all registered tools.
func NewToolsListCmd(reg *bootstrap.ToolRegistry) *cobra.Command {
	return &cobra.Command{
		Use:   "tools",
		Short: "List available tool subcommands",
		Long: `tools lists every external CLI tool that crank can wrap as a subcommand.

Each tool is available as: crank <tool> [args] --project <dir>
If --project is not specified, the current directory is used.
Missing tools are installed automatically when possible.

Examples:
  crank tools                        (list all tools)
  crank migrate up                   (run migrate in current directory)
  crank build --project ./myapp      (build a specific project)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Available tool subcommands:")
			fmt.Println()
			for _, t := range reg.All() {
				reqs := ""
				if r := t.RequiresFeatures(); len(r) > 0 {
					reqs = fmt.Sprintf(" (requires: %s)", strings.Join(r, ", "))
				}
				fmt.Printf("  %-12s %s%s\n", t.Name(), t.Description(), reqs)
			}
			fmt.Println()
			fmt.Println("Each tool is available as: crank <tool> [args] --project <dir>")
			fmt.Println("If --project is omitted, the current directory is used as the project root.")
			fmt.Println("Missing tools are installed automatically when possible.")
			return nil
		},
	}
}

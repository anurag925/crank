package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"

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

	cmd := &cobra.Command{
		Use:                t.Name(),
		Short:              t.Description(),
		Long:               t.LongDescription(),
		DisableFlagParsing: true, // We handle parsing manually to forward unknown flags.
		SilenceUsage:       true,
		SilenceErrors:      true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Extract --project and --project=<dir> from args.
			projectDir, rest := splitProjectFlag(args)

			// Register tool flags on cmd.Flags().
			cmd.Flags().String("project", ".", "path to the project directory")
			t.AddFlags(cmd)

			// Separate known flags (registered on cmd.Flags()) from
			// unknown flags and positional args that get forwarded.
			knownArgs, extraArgs := splitKnownUnknown(cmd.Flags(), rest)

			// Parse the known flags to set values on the tool's struct fields.
			if err := cmd.Flags().Parse(knownArgs); err != nil {
				return err
			}

			return executeTool(t, projectDir, cmd, extraArgs)
		},
	}

	return cmd, nil
}

// executeTool runs the full lifecycle: validate requirements → prepare → find binary → run.
func executeTool(t bootstrap.Tool, projectDir string, cmd *cobra.Command, extraArgs []string) error {
	// 1. Validate feature requirements against the project manifest.
	if err := bootstrap.ValidateToolRequirements(projectDir, t); err != nil {
		return err
	}

	// 2. In-process tools skip the binary lookup entirely. They run their
	// checks in-process and write results to stdout.
	if ip, ok := t.(bootstrap.InProcessTool); ok {
		results, err := ip.RunInProcess(projectDir, cmd.OutOrStdout())
		if err != nil {
			return err
		}
		for _, r := range results {
			if r.OK {
				fmt.Fprintf(cmd.OutOrStdout(), "✔  %s\n", r.Summary)
				continue
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✘  %s\n", r.Summary)
			if r.Detail != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "   %s\n", r.Detail)
			}
		}
		// Exit non-zero if any check failed.
		for _, r := range results {
			if !r.OK {
				return fmt.Errorf("doctor found %d issue(s)", countFailures(results))
			}
		}
		return nil
	}

	// 3. Let the tool build its invocation.
	inv, err := t.Prepare(projectDir, cmd, extraArgs)
	if err != nil {
		return err
	}

	// 4. Resolve the binary.
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

	// 5. Execute.
	fmt.Printf("→ %s %s\n", t.BinaryName(), strings.Join(inv.Args, " "))
	return utils.RunExternal(&utils.ExecConfig{
		Binary: inv.Binary,
		Args:   inv.Args,
		Dir:    inv.Dir,
		Env:    inv.Env,
		Stdin:  inv.Stdin,
	})
}

func countFailures(rs []bootstrap.CheckResult) int {
	n := 0
	for _, r := range rs {
		if !r.OK {
			n++
		}
	}
	return n
}

// splitKnownUnknown separates args into known (registered on the FlagSet) and
// unknown. Known flags and their values go into knownArgs; everything else
// (unknown flags, positional args) goes into extraArgs for forwarding.
func splitKnownUnknown(flags *flag.FlagSet, args []string) (knownArgs, extraArgs []string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]

		// End-of-flags marker: everything after goes to extra.
		if arg == "--" {
			extraArgs = append(extraArgs, args[i:]...)
			break
		}

		// Not a flag (positional arg, or just "-").
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			extraArgs = append(extraArgs, arg)
			continue
		}

		// Determine if it's a known flag.
		name, _, hasInlineValue := parseOneFlag(arg)
		var f *flag.Flag
		if strings.HasPrefix(arg, "--") {
			f = flags.Lookup(name)
		} else if len(name) == 1 {
			// Single-character shorthand (e.g. -v).
			f = flags.ShorthandLookup(name)
		} else {
			// Multi-character flag with single dash (e.g. -count).
			// pflag treats these as long flags.
			f = flags.Lookup(name)
		}

		if f != nil {
			// Known flag.
			knownArgs = append(knownArgs, arg)
			if !hasInlineValue && !isBoolFlag(f) && i+1 < len(args) {
				// Flag expects a value; consume the next arg.
				i++
				knownArgs = append(knownArgs, args[i])
			}
		} else {
			// Unknown flag → forward to the underlying tool.
			extraArgs = append(extraArgs, arg)
			if !hasInlineValue && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				// Looks like it takes a value; forward that too.
				i++
				extraArgs = append(extraArgs, args[i])
			}
		}
	}
	return
}

// parseOneFlag parses a flag argument like "--name=val", "--name", "-n=val",
// "-n", or "-nval". It returns the flag name, the inline value (if any), and
// whether an inline value was present.
func parseOneFlag(arg string) (name, value string, hasValue bool) {
	// Strip leading dashes.
	trimmed := strings.TrimLeft(arg, "-")

	if idx := strings.IndexByte(trimmed, '='); idx >= 0 {
		return trimmed[:idx], trimmed[idx+1:], true
	}
	return trimmed, "", false
}

// isBoolFlag reports whether f is a boolean flag (i.e. does not expect a
// separate value argument). Boolean flags set NoOptDefVal to "true" so the
// value can be omitted; in that case the next arg is not consumed.
func isBoolFlag(f *flag.Flag) bool {
	return f.NoOptDefVal != ""
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

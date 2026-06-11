package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/anurag925/rev/internal/bootstrap"
	"github.com/anurag925/rev/internal/utils"
)

// NewInitCmd returns the `init` cobra command.
func NewInitCmd(reg *bootstrap.Registry, toolReg *bootstrap.ToolRegistry) *cobra.Command {
	var (
		features string
		module   string
		target   string
		force    bool
	)

	cmd := &cobra.Command{
		Use:   "init <project>",
		Short: "Scaffold a new backend project",
		Long: `init creates a new directory named <project> inside the current working
directory and populates it with a production-ready Go backend. The base feature
is always included; additional modules are opt-in via --features.

After scaffolding, rev checks that all CLI tools required by the selected
features are installed and offers to install any that are missing.

Examples:
  rev init myapp --features=base,auth,postgres
  rev init myapp --module=github.com/org/myapp --features=base,redis`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			list := splitCSV(features)
			result, err := bootstrap.Generate(reg, bootstrap.Options{
				ProjectName: args[0],
				ModulePath:  module,
				TargetDir:   target,
				Features:    list,
				Force:       force,
			})
			if err != nil {
				return err
			}
			fmt.Printf("✔ Created %s with features: %s\n", result.ProjectDir, strings.Join(result.FeaturesUsed(), ", "))
			fmt.Printf("  %d files written.\n", len(result.Files))
			fmt.Println()

			// Check and install required tools for the enabled features.
			checkAndInstallTools(toolReg, result.Features)

			fmt.Println()
			fmt.Println("Next steps:")
			fmt.Println("  cd " + result.ProjectDir)
			fmt.Println("  rev tidy")
			fmt.Println("  rev run")
			return nil
		},
	}

	cmd.Flags().StringVar(&features, "features", "base", "comma-separated list of features to install (e.g. base,auth,postgres)")
	cmd.Flags().StringVar(&module, "module", "", "Go module path (defaults to the project name)")
	cmd.Flags().StringVar(&target, "target", ".", "parent directory in which to create the project")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing non-empty target directory")

	return cmd
}

// checkAndInstallTools verifies that the tools required by the enabled features
// are available on PATH, and installs them if they're missing.
func checkAndInstallTools(toolReg *bootstrap.ToolRegistry, features []string) {
	tools := toolReg.ForFeatures(features)
	if len(tools) == 0 {
		return
	}

	fmt.Println("Checking required tools...")

	var missing []bootstrap.Tool
	for _, t := range tools {
		_, err := utils.FindBinary(t.BinaryName(), "")
		if err != nil {
			missing = append(missing, t)
		} else {
			fmt.Printf("  ✔ %s found\n", t.BinaryName())
		}
	}

	if len(missing) == 0 {
		fmt.Println("  ✔ All required tools are installed.")
		return
	}

	fmt.Printf("\n  Missing %d tool(s). Installing...\n", len(missing))
	for _, t := range missing {
		if t.InstallCmd() == "" {
			fmt.Printf("  ⚠ %s not found and has no auto-install command. Please install manually.\n", t.BinaryName())
			continue
		}
		if err := t.Install(); err != nil {
			fmt.Printf("  ✗ Failed to install %s: %v\n  Manual install: %s\n", t.BinaryName(), err, t.InstallCmd())
		}
	}
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := parts[:0]
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

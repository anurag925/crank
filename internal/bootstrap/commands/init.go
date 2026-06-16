package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/anurag925/crank/internal/bootstrap"
	"github.com/anurag925/crank/internal/utils"
)

// NewInitCmd returns the `init` cobra command.
func NewInitCmd(reg *bootstrap.Registry, toolReg *bootstrap.ToolRegistry) *cobra.Command {
	var (
		features string
		module   string
		target   string
		force    bool
		useBun   bool
	)

	cmd := &cobra.Command{
		Use:   "init [project]",
		Short: "Scaffold a new backend project (interactive)",
		Long: `init creates a new directory named <project> inside the current working
directory and populates it with a production-ready Go backend. The base feature
is always included; additional modules are opt-in.

When run without explicit flags, init enters interactive mode and prompts you
for each setting step by step. Pass flags to skip the wizard and run
non-interactively.

By default, the project is scaffolded with GORM as the database ORM. Pass
--use-bun to use Bun (uptrace) instead. To skip the database entirely (e.g.
for an in-memory-only project) pass --features=base and crank will not add
any ORM feature.

Examples:
  crank init myapp --features=base,auth                       # gorm (default)
  crank init myapp --use-bun --features=base,auth             # bun
  crank init myapp --module=github.com/org/myapp --features=base,redis`,
		RunE: func(cmd *cobra.Command, args []string) error {
			interactive := shouldRunInteractive(cmd)

			var projectName string
			var featList []string

			if interactive {
				reader := bufio.NewReader(os.Stdin)

				// Step 1: Project name.
				if len(args) >= 1 {
					projectName = args[0]
					fmt.Printf("Project name: %s\n", projectName)
				} else {
					projectName = prompt(reader, "Project name")
					if projectName == "" {
						return fmt.Errorf("project name is required")
					}
				}

				// Step 2: Go module path.
				defaultModule := projectName
				if module != "" {
					defaultModule = module
				}
				moduleInput := prompt(reader, fmt.Sprintf("Go module path (default: %s)", defaultModule))
				if moduleInput == "" {
					moduleInput = defaultModule
				}
				module = moduleInput

				// Step 3: ORM choice. GORM is the default; bun is opt-in.
				defaultORM := "gorm"
				if useBun {
					defaultORM = "bun"
				}
				ormInput := prompt(reader, fmt.Sprintf("Default ORM (gorm/bun, default: %s)", defaultORM))
				switch strings.ToLower(strings.TrimSpace(ormInput)) {
				case "bun":
					useBun = true
				case "gorm", "":
					useBun = false
				default:
					return fmt.Errorf("unknown ORM %q (expected gorm or bun)", ormInput)
				}

				// Step 4: Select features.
				fmt.Println()
				fmt.Println("Available features:")
				allFeatures := reg.All()
				var optional []bootstrap.Feature
				for _, f := range allFeatures {
					if f.Name() == "base" || f.Name() == "gorm" || f.Name() == "bun" {
						// base is always-on; bun/gorm are added by the ORM step.
						continue
					}
					optional = append(optional, f)
					fmt.Printf("  [%d]      %s - %s\n", len(optional), f.Name(), f.Description())
				}
				fmt.Println()
				featInput := prompt(reader, "Select features (comma-separated numbers, e.g. 2,3 or 'all')")
				featList = parseFeatureSelection(featInput, optional)
				// base is always first; default ORM is added after.
				featList = append([]string{"base"}, featList...)
				if useBun {
					featList = append(featList, "bun")
				} else {
					featList = append(featList, "gorm")
				}

				// Step 5: Target directory.
				defaultTarget := target
				if defaultTarget == "" {
					defaultTarget = "."
				}
				targetInput := prompt(reader, fmt.Sprintf("Target directory (default: %s)", defaultTarget))
				if targetInput != "" {
					target = targetInput
				}

				// Step 6: Force overwrite.
				projectDir := filepath.Join(target, projectName)
				if utils.PathExists(projectDir) && !force {
					force = confirm(reader, "Directory already exists. Overwrite")
				}
			} else {
				if len(args) < 1 {
					return fmt.Errorf("project name is required (run 'crank init' without flags for interactive mode)")
				}
				projectName = args[0]
				featList = splitCSV(features)
				// Ensure base is included, then auto-add the default ORM if the
				// user did not explicitly pick one. Passing --use-bun flips the
				// default to bun.
				featList = ensureBase(featList)
				featList = ensureDefaultORM(featList, useBun)
			}

			result, err := bootstrap.Generate(reg, bootstrap.Options{
				ProjectName: projectName,
				ModulePath:  module,
				TargetDir:   target,
				Features:    featList,
				Force:       force,
			})
			if err != nil {
				return err
			}
			fmt.Printf("✔ Created %s with features: %s\n", result.ProjectDir, strings.Join(result.FeaturesUsed(), ", "))
			fmt.Printf("  %d files written.\n", len(result.Files))
			fmt.Println()

			// Install dependencies via go get.
			if err := bootstrap.GoGet(result.ProjectDir, result.Dependencies); err != nil {
				return fmt.Errorf("install dependencies: %w", err)
			}
			fmt.Println()

			// Check and install required tools for the enabled features.
			checkAndInstallTools(toolReg, result.Features)

			fmt.Println()
			fmt.Println("Next steps:")
			fmt.Println("  cd " + result.ProjectDir)
			fmt.Println("  crank run")
			return nil
		},
	}

	cmd.Flags().StringVar(&features, "features", "base", "comma-separated list of features to install (e.g. base,auth,redis). GORM is added by default unless --use-bun is passed or another ORM is in the list.")
	cmd.Flags().StringVar(&module, "module", "", "Go module path (defaults to the project name)")
	cmd.Flags().StringVar(&target, "target", ".", "parent directory in which to create the project")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing non-empty target directory")
	cmd.Flags().BoolVar(&useBun, "use-bun", false, "use Bun (uptrace) as the default ORM instead of GORM")

	return cmd
}

// ensureDefaultORM injects a default ORM into the feature list when none is
// present. GORM is the default; --use-bun flips it to bun. Calling with an
// explicit ORM in the list is a no-op so the user can override the default
// (e.g. `crank init myapp --features=base,bun`).
func ensureDefaultORM(features []string, useBun bool) []string {
	hasBun := false
	hasGorm := false
	for _, f := range features {
		if f == "bun" {
			hasBun = true
		}
		if f == "gorm" {
			hasGorm = true
		}
	}
	if hasBun || hasGorm {
		return features
	}
	if useBun {
		return append(features, "bun")
	}
	return append(features, "gorm")
}

// ensureBase is a defensive helper for the init command (the generator also
// does this internally, but we want a stable feature list early so the
// interactive prompt and the default-ORM logic can reason about it).
func ensureBase(features []string) []string {
	for _, f := range features {
		if f == "base" {
			return features
		}
	}
	out := make([]string, 0, len(features)+1)
	out = append(out, "base")
	out = append(out, features...)
	return out
}

// shouldRunInteractive decides whether to run the interactive wizard.
// It returns true when no flags are explicitly set and stdin is a terminal.
func shouldRunInteractive(cmd *cobra.Command) bool {
	if cmd.Flags().Changed("features") ||
		cmd.Flags().Changed("module") ||
		cmd.Flags().Changed("target") ||
		cmd.Flags().Changed("force") ||
		cmd.Flags().Changed("use-bun") {
		return false
	}
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// prompt prints a question and reads a line from the user.
func prompt(reader *bufio.Reader, question string) string {
	fmt.Printf("%s: ", question)
	text, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}
	return strings.TrimSpace(text)
}

// confirm asks a yes/no question and returns true if the answer is "y" or "yes".
func confirm(reader *bufio.Reader, question string) bool {
	answer := prompt(reader, question+" (y/N)")
	lower := strings.ToLower(answer)
	return lower == "y" || lower == "yes"
}

// parseFeatureSelection converts a user input like "1,3" or "all" into a list
// of feature names using the provided ordered feature slice.
func parseFeatureSelection(input string, features []bootstrap.Feature) []string {
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return nil
	}
	if input == "all" {
		var out []string
		for _, f := range features {
			out = append(out, f.Name())
		}
		return out
	}
	var out []string
	parts := strings.Split(input, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > len(features) {
			fmt.Printf("  ⚠ Ignoring invalid selection: %q\n", p)
			continue
		}
		out = append(out, features[n-1].Name())
	}
	return out
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

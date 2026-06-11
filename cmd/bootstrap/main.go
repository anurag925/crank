package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/anurag925/rev/internal/bootstrap"
	"github.com/anurag925/rev/internal/bootstrap/commands"

	// Feature packages self-register with the global registry via init().
	_ "github.com/anurag925/rev/internal/bootstrap/features/auth"
	_ "github.com/anurag925/rev/internal/bootstrap/features/base"
	_ "github.com/anurag925/rev/internal/bootstrap/features/crypto"
	_ "github.com/anurag925/rev/internal/bootstrap/features/mongodb"
	_ "github.com/anurag925/rev/internal/bootstrap/features/postgres"
	_ "github.com/anurag925/rev/internal/bootstrap/features/redis"

	// Tool packages self-register with the global tool registry via init().
	_ "github.com/anurag925/rev/internal/bootstrap/tools/build"
	_ "github.com/anurag925/rev/internal/bootstrap/tools/dev"
	_ "github.com/anurag925/rev/internal/bootstrap/tools/gofmt"
	_ "github.com/anurag925/rev/internal/bootstrap/tools/migrate"
	_ "github.com/anurag925/rev/internal/bootstrap/tools/run"
	_ "github.com/anurag925/rev/internal/bootstrap/tools/swag"
	_ "github.com/anurag925/rev/internal/bootstrap/tools/test"
	_ "github.com/anurag925/rev/internal/bootstrap/tools/tidy"
	_ "github.com/anurag925/rev/internal/bootstrap/tools/vet"
)

// Build information, populated at release time via -ldflags by GoReleaser.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	root := newRootCmd()

	// If the first argument is not a native rev command but matches a target in
	// the target project's Makefile, transparently delegate to `make <target>`.
	// Native rev commands always take precedence.
	if handled, err := commands.TryMakeDelegation(root, os.Args[1:]); handled {
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		return
	}

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	featReg := bootstrap.GlobalRegistry
	toolReg := bootstrap.GlobalToolRegistry

	root := &cobra.Command{
		Use:     "rev",
		Version: fmt.Sprintf("%s (commit %s, built %s)", version, commit, date),
		Short:   "Scaffold and manage production-ready Go backend services",
		Long: `rev is a code generator and project management tool for Go backend
services. It scaffolds projects with a curated set of installable features and
wraps common CLI tools (go, migrate, swag, air, ...) as subcommands so you
never need to leave the rev CLI.

All tool subcommands accept --project to target a specific project directory.
If --project is not specified, the current directory is used.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Feature lifecycle commands.
	root.AddCommand(
		commands.NewInitCmd(featReg, toolReg),
		commands.NewAddCmd(featReg),
		commands.NewListCmd(featReg),
		commands.NewMakeCmd(),
		commands.NewToolsListCmd(toolReg),
	)

	// Register a cobra subcommand for every tool in the registry.
	for _, t := range toolReg.All() {
		toolCmd, err := commands.NewToolCmd(toolReg, t.Name())
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not register tool command %q: %v\n", t.Name(), err)
			continue
		}
		root.AddCommand(toolCmd)
	}

	return root
}

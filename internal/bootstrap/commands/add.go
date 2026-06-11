package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/anurag925/rev/internal/bootstrap"
)

// NewAddCmd returns the `add` cobra command.
func NewAddCmd(reg *bootstrap.Registry) *cobra.Command {
	var projectDir string

	cmd := &cobra.Command{
		Use:   "add <feature>",
		Short: "Install a feature into an existing project",
		Long: `add copies the templates of a single feature into a project previously
created with ` + "`init`" + `. The project must contain a .bootstrap.yaml manifest
so the bootstrapper knows which module path and feature set to use.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := bootstrap.Add(reg, projectDir, args[0])
			if err != nil {
				return err
			}
			fmt.Printf("✔ Installed %q into %s (%d files written)\n", args[0], result.ProjectDir, len(result.Files))
			for _, f := range result.Files {
				fmt.Println("    + " + f)
			}
			fmt.Println()
			fmt.Println("Don't forget to run `go mod tidy` to pull in any new dependencies.")
			return nil
		},
	}

	cmd.Flags().StringVar(&projectDir, "project", ".", "path to the project directory")
	return cmd
}

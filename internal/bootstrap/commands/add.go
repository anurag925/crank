package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/anurag925/crank/internal/bootstrap"
)

// NewAddCmd returns the `add` cobra command.
func NewAddCmd(reg *bootstrap.Registry) *cobra.Command {
	var projectDir string

	cmd := &cobra.Command{
		Use:   "add <feature>",
		Short: "Install a feature into an existing project",
		Long: `add copies the templates of a single feature into a project previously
created with ` + "`init`" + `. The project must contain a .crank.yaml manifest
so crank knows which module path and feature set to use.`,
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

			// Install the new feature's dependencies via go get.
			if err := bootstrap.GoGet(result.ProjectDir, result.Dependencies); err != nil {
				return fmt.Errorf("install dependencies: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&projectDir, "project", ".", "path to the project directory")
	return cmd
}

package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/anurag925/crank/internal/bootstrap"
)

// NewUpdateCmd returns the `update` cobra command.
func NewUpdateCmd() *cobra.Command {
	var projectDir string

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a crank project to the latest templates and tooling",
		Long: `update reconciles a crank-generated project with the latest version of the
crank CLI. It re-renders feature templates, injects updated config sections,
and bumps the crank_version in .crank.yaml.

Currently this command only updates the crank_version stamp in .crank.yaml.
Full template reconciliation will be added in a future release.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := bootstrap.Update(projectDir)
			if err != nil {
				return err
			}
			fmt.Printf("✔ Updated %s to crank version %s\n", result.ProjectDir, bootstrap.Version)
			return nil
		},
	}

	cmd.Flags().StringVar(&projectDir, "project", ".", "path to the project directory")
	return cmd
}

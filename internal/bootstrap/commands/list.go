package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/anurag925/crank/internal/bootstrap"
)

// NewListCmd returns the `list` cobra command.
func NewListCmd(reg *bootstrap.Registry) *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available features",
		RunE: func(cmd *cobra.Command, args []string) error {
			features := reg.All()
			if asJSON {
				type entry struct {
					Name        string `json:"name"`
					Description string `json:"description"`
				}
				out := make([]entry, 0, len(features))
				for _, f := range features {
					out = append(out, entry{Name: f.Name(), Description: f.Description()})
				}
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "NAME\tDESCRIPTION")
			for _, f := range features {
				fmt.Fprintf(tw, "%s\t%s\n", f.Name(), f.Description())
			}
			return tw.Flush()
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	return cmd
}

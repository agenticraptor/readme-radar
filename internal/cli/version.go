package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/agenticraptor/readme-radar/internal/buildinfo"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			fmt.Println(buildinfo.String())
			return nil
		},
	}
}

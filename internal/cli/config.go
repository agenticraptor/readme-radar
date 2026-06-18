package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/agenticraptor/readme-radar/internal/config"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config <init|path|show>",
		Short: "Manage the optional configuration file",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "init",
			Short: "Write a documented starter config (if none exists)",
			Args:  cobra.NoArgs,
			RunE: func(*cobra.Command, []string) error {
				path, err := config.Init()
				if err != nil {
					return err
				}
				fmt.Printf("config ready at %s\n", path)
				return nil
			},
		},
		&cobra.Command{
			Use:   "path",
			Short: "Print the config file path",
			Args:  cobra.NoArgs,
			RunE: func(*cobra.Command, []string) error {
				path, err := config.Path()
				if err != nil {
					return err
				}
				fmt.Println(path)
				return nil
			},
		},
		&cobra.Command{
			Use:   "show",
			Short: "Print the effective configuration",
			Args:  cobra.NoArgs,
			RunE: func(*cobra.Command, []string) error {
				cfg, err := config.Load()
				if err != nil {
					return err
				}
				fmt.Printf("[ai]\nprovider = %q\nmodel    = %q\nenabled  = %v\n", cfg.AI.Provider, cfg.AI.Model, cfg.AI.Enabled)
				if path, _ := config.Path(); path != "" {
					if _, statErr := os.Stat(path); statErr != nil {
						fmt.Printf("\n(no file yet — run `readme-radar config init`)\n")
					}
				}
				return nil
			},
		},
	)
	return cmd
}

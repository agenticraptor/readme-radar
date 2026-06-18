// Package cli wires the readme-radar command-line interface together.
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/agenticraptor/readme-radar/internal/buildinfo"
)

// Execute runs the root command and returns a process exit code.
func Execute() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := newRootCmd().ExecuteContext(ctx)
	if err == nil {
		return 0
	}
	var ce codeError
	if errors.As(err, &ce) {
		if ce.msg != "" {
			fmt.Fprintln(os.Stderr, "error:", ce.msg)
		}
		return ce.code
	}
	fmt.Fprintln(os.Stderr, "error:", err)
	return 1
}

// codeError carries a specific exit code (used for CI gating with --min-score).
type codeError struct {
	code int
	msg  string
}

func (e codeError) Error() string { return e.msg }

func newRootCmd() *cobra.Command {
	var o scanOptions

	cmd := &cobra.Command{
		Use:   "readme-radar <repo-or-package>",
		Short: "A 30-second \"is this safe & worth it\" verdict for any repo or package",
		Long: `readme-radar gives you a fast, shareable trust verdict for a GitHub repository
or an npm/PyPI package — the moment before you install a stranger's code.

It checks how actively the project is maintained, what it would cost to install
(size + dependencies), whether it phones home (analytics/telemetry), and the
scariest capabilities it asks for (install scripts, native builds), then prints
a single graded verdict card. Add an API key for a plain-English summary, or run
fully offline.

Examples:
  readme-radar charmbracelet/bubbletea
  readme-radar https://github.com/expressjs/express
  readme-radar npm:left-pad
  readme-radar pypi:requests --format markdown
  readme-radar react --card react.png`,
		Args:          cobra.ArbitraryArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       buildinfo.Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return runScan(cmd.Context(), args[0], o)
		},
	}
	addScanFlags(cmd, &o)
	cmd.SetVersionTemplate("{{.Name}} {{.Version}}\n")

	cmd.AddCommand(
		newScanCmd(),
		newCardCmd(),
		newConfigCmd(),
		newDoctorCmd(),
		newVersionCmd(),
	)
	return cmd
}

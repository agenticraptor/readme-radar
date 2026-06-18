package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/agenticraptor/readme-radar/internal/analyze"
	"github.com/agenticraptor/readme-radar/internal/config"
	"github.com/agenticraptor/readme-radar/internal/github"
	"github.com/agenticraptor/readme-radar/internal/llm"
	"github.com/agenticraptor/readme-radar/internal/narrate"
	"github.com/agenticraptor/readme-radar/internal/render"
	"github.com/agenticraptor/readme-radar/internal/report"
	"github.com/agenticraptor/readme-radar/internal/source"
)

type scanOptions struct {
	format   string
	output   string
	card     string
	scale    int
	noAI     bool
	provider string
	model    string
	token    string
	noColor  bool
	quiet    bool
	minScore int
}

func addScanFlags(cmd *cobra.Command, o *scanOptions) {
	f := cmd.Flags()
	f.StringVarP(&o.format, "format", "f", "", "Output format: term | plain | markdown | json (default: auto)")
	f.StringVarP(&o.output, "output", "o", "", "Write the report to a file instead of stdout")
	f.StringVar(&o.card, "card", "", "Also write a shareable verdict card to this path (.svg or .png)")
	f.IntVar(&o.scale, "scale", 2, "PNG card scale factor (1 = 860×470)")
	f.BoolVar(&o.noAI, "no-ai", false, "Skip the model; use the deterministic offline summary")
	f.StringVar(&o.provider, "provider", "", "LLM provider: anthropic | openai | ollama")
	f.StringVar(&o.model, "model", "", "Model name (defaults to the provider's default)")
	f.StringVar(&o.token, "token", "", "GitHub token (overrides GITHUB_TOKEN/GH_TOKEN)")
	f.BoolVar(&o.noColor, "no-color", false, "Disable colored output")
	f.BoolVarP(&o.quiet, "quiet", "q", false, "Suppress progress messages")
	f.IntVar(&o.minScore, "min-score", 0, "Exit non-zero if the overall score is below this (0-100); for CI")
}

func newScanCmd() *cobra.Command {
	var o scanOptions
	cmd := &cobra.Command{
		Use:           "scan <repo-or-package>",
		Short:         "Scan a repository or package (same as running with no subcommand)",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScan(cmd.Context(), args[0], o)
		},
	}
	addScanFlags(cmd, &o)
	return cmd
}

func runScan(ctx context.Context, arg string, o scanOptions) error {
	rep, err := scan(ctx, arg, o)
	if err != nil {
		return err
	}

	format := resolveFormat(o, os.Stdout)
	out, err := render.Render(rep, render.Options{Format: render.Format(format), NoColor: o.noColor || !isTTY(os.Stdout)})
	if err != nil {
		return err
	}

	if o.output != "" {
		if err := os.WriteFile(o.output, []byte(out+"\n"), 0o644); err != nil {
			return err
		}
		progress(o, "wrote %s", o.output)
	} else {
		fmt.Println(out)
	}

	if o.card != "" {
		if err := writeCard(rep, o.card, o.scale); err != nil {
			return fmt.Errorf("write card: %w", err)
		}
		progress(o, "wrote card %s", o.card)
	}

	if o.minScore > 0 && rep.Score.Overall < o.minScore {
		return codeError{code: 2, msg: fmt.Sprintf("%s scored %d, below the --min-score threshold of %d",
			rep.Target.Slug(), rep.Score.Overall, o.minScore)}
	}
	return nil
}

// scan resolves, analyzes, and narrates a target into a Report.
func scan(ctx context.Context, arg string, o scanOptions) (report.Report, error) {
	target, err := source.Parse(arg)
	if err != nil {
		return report.Report{}, err
	}

	az := analyze.New()
	if o.token != "" {
		az.GH.Token = o.token
	}

	progress(o, "scanning %s …", target.Slug())
	// Bound the network-bound analysis so a scan can never hang indefinitely.
	actx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	rep, err := az.Analyze(actx, target)
	if err != nil {
		var rl *github.RateLimitError
		if errors.As(err, &rl) {
			return rep, fmt.Errorf("%w\n  tip: set GITHUB_TOKEN (a classic token with no scopes works) to raise the limit", rl)
		}
		return rep, err
	}

	rep.Narrative = narrate.Narrate(ctx, rep, resolveClient(o))
	return rep, nil
}

// resolveClient builds an LLM client from flags/config/env, or returns nil to
// use the offline narrative. It never returns an error: narration is optional.
func resolveClient(o scanOptions) llm.Client {
	if o.noAI {
		return nil
	}
	cfg, _ := config.Load()
	if !cfg.AI.Enabled {
		return nil
	}
	provider := first(o.provider, cfg.AI.Provider)
	model := first(o.model, cfg.AI.Model)

	if provider != "" {
		c, err := llm.New(provider, model, "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "note: %v — using offline summary\n", err)
			return nil
		}
		return c
	}
	c, err := llm.Detect()
	if err != nil {
		return nil
	}
	return c
}

func resolveFormat(o scanOptions, f *os.File) string {
	if o.format != "" {
		return o.format
	}
	if o.output != "" {
		return string(render.FormatPlain)
	}
	if isTTY(f) {
		return string(render.FormatTerm)
	}
	return string(render.FormatPlain)
}

func isTTY(f *os.File) bool {
	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}

func progress(o scanOptions, format string, args ...any) {
	if o.quiet || o.format == "json" {
		return
	}
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func first(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

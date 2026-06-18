package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/agenticraptor/readme-radar/internal/card"
	"github.com/agenticraptor/readme-radar/internal/narrate"
	"github.com/agenticraptor/readme-radar/internal/report"
)

func newCardCmd() *cobra.Command {
	var o scanOptions
	cmd := &cobra.Command{
		Use:           "card <repo-or-package>",
		Short:         "Render a shareable verdict card (SVG or PNG)",
		Long:          "Scan a target and write a shareable verdict card. The format is chosen from the\noutput extension (.svg or .png). With no --output, a PNG is written next to you.",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			rep, err := scan(cmd.Context(), args[0], o)
			if err != nil {
				return err
			}
			path := o.output
			if path == "" {
				path = defaultCardName(rep.Target)
			}
			if err := writeCard(rep, path, o.scale); err != nil {
				return err
			}
			progress(o, "wrote %s", path)
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVarP(&o.output, "output", "o", "", "Output path (.svg or .png; default: <name>-radar.png)")
	f.IntVar(&o.scale, "scale", 2, "PNG scale factor (1 = 860×470)")
	f.BoolVar(&o.noAI, "no-ai", false, "Skip the model; use the deterministic offline summary")
	f.StringVar(&o.provider, "provider", "", "LLM provider: anthropic | openai | ollama")
	f.StringVar(&o.model, "model", "", "Model name")
	f.StringVar(&o.token, "token", "", "GitHub token (overrides GITHUB_TOKEN/GH_TOKEN)")
	f.BoolVarP(&o.quiet, "quiet", "q", false, "Suppress progress messages")
	return cmd
}

// writeCard renders the verdict card to path, choosing SVG or PNG by extension.
func writeCard(rep report.Report, path string, scale int) error {
	if rep.Narrative.Summary == "" {
		rep.Narrative = narrate.Narrate(context.Background(), rep, nil)
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".svg":
		return os.WriteFile(path, []byte(card.SVG(rep)), 0o644)
	case ".png", "":
		if filepath.Ext(path) == "" {
			path += ".png"
		}
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		return card.PNG(rep, f, scale)
	default:
		return fmt.Errorf("unsupported card extension %q (use .svg or .png)", filepath.Ext(path))
	}
}

func defaultCardName(t report.Target) string {
	base := t.Repo
	if base == "" {
		base = t.Package
	}
	if base == "" {
		base = "readme-radar"
	}
	base = strings.NewReplacer("/", "-", "@", "", " ", "-").Replace(base)
	return base + "-radar.png"
}

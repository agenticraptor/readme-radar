package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/agenticraptor/readme-radar/internal/config"
	"github.com/agenticraptor/readme-radar/internal/github"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check your environment and configuration",
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			runDoctor()
			return nil
		},
	}
}

func runDoctor() {
	ok := func(label, detail string) { fmt.Printf("  \033[32m✓\033[0m %-18s %s\n", label, detail) }
	warn := func(label, detail string) { fmt.Printf("  \033[33m!\033[0m %-18s %s\n", label, detail) }

	fmt.Print("readme-radar doctor\n\n")

	gh := github.New()
	if gh.Authenticated() {
		ok("github token", "set — API limit is 5,000 requests/hour")
	} else {
		warn("github token", "none — limited to 60 requests/hour. Set GITHUB_TOKEN to raise it")
	}

	switch {
	case os.Getenv("ANTHROPIC_API_KEY") != "":
		ok("ai provider", "anthropic (ANTHROPIC_API_KEY set)")
	case os.Getenv("OPENAI_API_KEY") != "":
		ok("ai provider", "openai (OPENAI_API_KEY set)")
	default:
		warn("ai provider", "no API key — will try local Ollama, or use --no-ai for the offline summary")
	}

	if path, err := config.Path(); err == nil {
		if _, statErr := os.Stat(path); statErr == nil {
			ok("config", path)
		} else {
			warn("config", "none yet ("+path+") — run `readme-radar config init` to create one")
		}
	}

	fmt.Print("\nNetwork: readme-radar contacts api.github.com (and the npm/PyPI registries),\nplus your chosen LLM provider only when narration is enabled.\n")
	fmt.Print("\nTip: run `readme-radar <owner/repo>` — e.g. `readme-radar charmbracelet/bubbletea`.\n")
}

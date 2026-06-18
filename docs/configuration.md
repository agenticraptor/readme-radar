# Configuration

Configuration is **optional** — readme-radar runs with no config at all. A file
only lets you set defaults for the AI narration so you don't repeat flags.

## Location

```bash
readme-radar config path     # print the path
readme-radar config init     # write a documented starter (if none exists)
readme-radar config show     # print the effective settings
```

The file lives at `~/.config/readme-radar/config.toml` (or
`$XDG_CONFIG_HOME/readme-radar/config.toml` if that variable is set).

## Format

```toml
[ai]
# Provider for the verdict narration.
#   anthropic | openai | ollama   (empty = auto-detect from your environment)
provider = ""
# Model name (empty = the provider's default).
model = ""
# Set false to always use the deterministic offline narrative.
enabled = true
```

## Environment variables

Secrets are **never** stored in the config file — they are read from the
environment:

| Variable | Purpose |
|----------|---------|
| `GITHUB_TOKEN` / `GH_TOKEN` | Raises the GitHub API limit from 60 to 5,000 req/hr |
| `ANTHROPIC_API_KEY` | Enables Anthropic narration |
| `OPENAI_API_KEY` | Enables OpenAI narration |
| `README_RADAR_MODEL` | Overrides the model name |
| `OLLAMA_HOST` | Points at a non-default Ollama endpoint |

## Precedence

For any setting, the order is: **command-line flag** → **config file** →
**environment** → built-in default. For example, `--model` beats `[ai].model`,
which beats `README_RADAR_MODEL`, which beats the provider's default.

## CI usage

`--min-score` makes readme-radar exit non-zero when a target scores below a
threshold — handy as a dependency gate:

```bash
readme-radar npm:some-new-dep --no-ai --min-score 60 || echo "below the bar"
```

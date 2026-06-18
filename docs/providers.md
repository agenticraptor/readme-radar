# AI providers

The plain-English verdict is written by a model when one is available, and by a
deterministic offline summary otherwise. **Narration is optional and never
affects the grade** — readme-radar is fully useful with `--no-ai`.

## Auto-detection

With no provider configured, readme-radar picks one from your environment:

1. **Anthropic** — if `ANTHROPIC_API_KEY` is set.
2. **OpenAI** — if `OPENAI_API_KEY` is set.
3. **Ollama** — otherwise (local; falls back to the offline summary if it isn't
   running).

Override the model anywhere with the `README_RADAR_MODEL` environment variable,
the `--model` flag, or the config file.

## Anthropic

```bash
export ANTHROPIC_API_KEY=sk-ant-...
readme-radar charmbracelet/bubbletea
readme-radar charmbracelet/bubbletea --model claude-sonnet-4-6
```

Default model: `claude-sonnet-4-6`.

## OpenAI

```bash
export OPENAI_API_KEY=sk-...
readme-radar expressjs/express --provider openai
readme-radar expressjs/express --provider openai --model gpt-4o
```

Default model: `gpt-4o-mini`.

## Ollama (fully local)

Run [Ollama](https://ollama.com) and point readme-radar at it — nothing leaves
your machine:

```bash
ollama serve            # if it isn't already running
ollama pull llama3.1
readme-radar npm:express --provider ollama --model llama3.1
```

Default model: `llama3.1`. Set `OLLAMA_HOST` if it listens somewhere other than
`http://localhost:11434`.

## Offline

```bash
readme-radar npm:left-pad --no-ai
```

The offline summary is assembled from the same facts (verdict headline,
maintenance status, footprint, install scripts, telemetry). It's always
available, always free, and always local.

## What gets sent

When a **cloud** model is selected, readme-radar sends a short, structured
**facts brief** of the *already-computed* report (grades, counts, factor text) so
the model can phrase it. It does not send your code or credentials. Use
`--no-ai` or `--provider ollama` to keep everything local. See
[../SECURITY.md](../SECURITY.md) for the full data-handling notes.

# Contributing to readme-radar

Thanks for your interest in contributing! This project aims to be a small,
focused, dependency-light tool — contributions that keep it that way are
especially appreciated.

## Getting started

```bash
git clone https://github.com/agenticraptor/readme-radar
cd readme-radar
go mod tidy        # fetch dependencies & populate go.sum
make build         # build into ./bin/readme-radar
make test          # run the unit tests
./bin/readme-radar charmbracelet/bubbletea --no-ai
```

Requirements:

- Go 1.22 or newer
- (optional) a `GITHUB_TOKEN` to avoid the unauthenticated 60 req/hr API limit
- (optional) [`golangci-lint`](https://golangci-lint.run/) for `make lint`
- (optional) [`goreleaser`](https://goreleaser.com/) for `make snapshot`

## Development workflow

1. Fork the repo and create a feature branch from `main`.
2. Make your change, with tests where it makes sense.
3. Run the full check suite locally:
   ```bash
   make fmt vet test
   ```
4. Open a pull request. Fill in the PR template and link any related issue.

CI runs `gofmt`, `go vet`, `golangci-lint`, and the test suite on Linux, macOS,
and Windows. All checks must pass before review.

## Commit messages

We use [Conventional Commits](https://www.conventionalcommits.org/). This keeps
the generated changelog readable and drives semantic-version bumps.

```
feat: detect Cargo crates via crates.io
fix: handle scoped npm packages with no repository field
docs: clarify the scoring weights
test: cover the deprecation gate
chore: bump lipgloss
```

## Coding guidelines

- **Keep dependencies minimal.** readme-radar ships with only a handful of
  direct dependencies. Prefer the standard library; if a new dependency is truly
  needed, call it out in the PR description.
- **Deterministic first.** Every score must be reproducible from the same
  inputs. The model writes prose; it must **never** influence a number or a
  grade. Adding a signal means adding it to the relevant analyzer, not the
  prompt.
- **Tolerant parsing.** Anything that reads external data (GitHub/registry JSON,
  manifests, model responses) must degrade gracefully — skip the bad record,
  attach a warning, never crash the scan.
- **The offline path is sacred.** The tool must remain fully useful with no API
  key. If you touch narration, make sure `--no-ai` still produces a good result.
- **Honest claims only.** A clean telemetry/permissions result is *best-effort*,
  not a guarantee. Don't word findings more strongly than the evidence supports.
- **Format with `gofmt -s`** and keep `go vet` clean.

## Architecture at a glance

| Package | Responsibility |
|---------|----------------|
| `internal/source` | Parse user input → a resolved `Target` (no network). |
| `internal/github` | Tiny REST client for the repo/contents/contributors/releases endpoints. |
| `internal/registry` | npm + PyPI clients: install size, deps, scripts, deprecation, repo URL. |
| `internal/analyze` | Orchestrate fetch → analyzers → combined `Report`. |
| `internal/health` | Maintenance scoring from GitHub facts. |
| `internal/telemetry` | Analytics/telemetry detection from deps + manifest. |
| `internal/permissions` | Install scripts, native builds, process spawning. |
| `internal/score` | Combine axes into the overall grade, with the gating rules. |
| `internal/llm` | HTTP clients for Anthropic, OpenAI, and Ollama. |
| `internal/narrate` | LLM narration + the deterministic offline summary. |
| `internal/render` | Render a `Report` to term / plain / Markdown / JSON. |
| `internal/card` | The shareable verdict card (SVG + PNG). |
| `internal/cli` | Cobra commands. |

## Good first issues

- Add a **Cargo** (`crates.io`) or **Go module** footprint source.
- Expand the analytics/telemetry SDK lists (`internal/telemetry`).
- Add more risk heuristics to `internal/permissions` (e.g. `obfuscated` code,
  base64-decoded `eval`).
- A `--watch`/badge mode that emits an SVG for embedding in a README.
- More dependency ecosystems for deprecation/yank detection.

## Reporting bugs & requesting features

Use the [issue templates](https://github.com/agenticraptor/readme-radar/issues/new/choose).
For anything security-related, please follow [SECURITY.md](SECURITY.md) instead
of opening a public issue.

## License

By contributing, you agree that your contributions will be licensed under the
[MIT License](LICENSE).

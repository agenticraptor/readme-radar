<div align="center">

# 📡 readme-radar

### Drop a repo or package name, get the 30-second *"is this safe & worth it?"* verdict — before you install it.

`readme-radar` scans any GitHub repository or npm/PyPI package and prints one
graded **verdict card**: how actively it's maintained, what it costs to install
(size + dependencies), whether it phones home, and the scariest things it does at
install time — plus a plain-English *"what is it & should you trust it."*

The moment before you `npm i` a stranger's package, paste it here instead.

[![CI](https://github.com/agenticraptor/readme-radar/actions/workflows/ci.yml/badge.svg)](https://github.com/agenticraptor/readme-radar/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/agenticraptor/readme-radar?sort=semver)](https://github.com/agenticraptor/readme-radar/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/agenticraptor/readme-radar.svg)](https://pkg.go.dev/github.com/agenticraptor/readme-radar)
[![Go Report Card](https://goreportcard.com/badge/github.com/agenticraptor/readme-radar)](https://goreportcard.com/report/github.com/agenticraptor/readme-radar)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

</div>

---

> **Try it in one line — no install, no signup, no key:**
>
> ```bash
> go run github.com/agenticraptor/readme-radar/cmd/readme-radar@latest charmbracelet/bubbletea
> ```
>
> Add `--no-ai` for a fully offline run, or set `ANTHROPIC_API_KEY` /
> `OPENAI_API_KEY` (or point it at a local Ollama) for the plain-English verdict.

<!--
  📸 Replace this block with a 15–20s screen-capture GIF: paste an npm package,
  watch the graded card appear, then `--card out.png` to export it. The hero GIF
  is the single biggest driver of stars — record it once, drop it at
  docs/demo.gif, then uncomment:

  <p align="center"><img src="docs/demo.gif" alt="readme-radar demo" width="760"></p>
-->

```text
╭────────────────────────────────────────────────────────────────╮
│  📡 readme-radar                                  npm:express   │
│                                                                │
│  [ A ]  LOOKS TRUSTWORTHY                            98 / 100   │
│                                                                │
│  Maintenance  A  ██████████  100                               │
│  Permissions  A  ██████████  100                               │
│  Telemetry    A  ██████████  100                               │
│  Footprint    A  █████████░  88                                │
│                                                                │
│  ★ 69,189   ⬇ 73.7 KB   ⬚ 28 deps   ⚖ MIT   ⟳ today            │
│                                                                │
│  expressjs/express — Fast, unopinionated, minimalist web       │
│  framework for node. Healthy, lean, and quiet — safe to adopt  │
│  with the usual review.                                        │
│  • No install scripts                                          │
╰────────────────────────────────────────────────────────────────╯
```

…and the one you actually need to see:

```text
╭────────────────────────────────────────────────────────────────╮
│  📡 readme-radar                                  npm:request   │
│                                                                │
│  [ D ]  PROCEED WITH CAUTION                         50 / 100   │
│                                                                │
│  Maintenance  F  ████░░░░░░  35                                │
│  Permissions  A  ██████████  100                               │
│  Telemetry    A  ██████████  100                               │
│  Footprint    B  ████████░░  78                                │
│                                                                │
│  • Maintenance F: request has been deprecated, see #3142       │
╰────────────────────────────────────────────────────────────────╯
```

## Why

Installing a dependency in 2026 is an act of trust. You're about to run a
stranger's code on your machine and ship it to your users — but the signals you'd
want are scattered across six tabs: the repo's last commit, its bundle size on
one site, its install scripts buried in `package.json`, whether it's been
quietly deprecated. So most people skip the check and hope.

readme-radar collapses that into **one paste → one verdict**:

- **A real recommendation, not a data dump.** A single A–F grade and a
  *trust / caution / avoid* verdict, backed by four scored axes you can scan in
  two seconds.
- **It catches the things that bite you.** A `curl … | bash` postinstall, an
  embedded analytics SDK, a package the author **deprecated** months ago — each
  *caps* the verdict so a popular repo can't hide it behind its star count.
- **Real numbers.** Actual unpacked install size and dependency counts from the
  npm/PyPI registries — the falsifiable kind that survives a retell ("73.7 KB,
  28 deps").
- **A card you can share.** Export the verdict as an **SVG or PNG** and drop it
  in an issue, a PR, or a tweet.

## Features

- 🎯 **One graded verdict** (A–F + trust/caution/avoid) from four scored axes:
  **Maintenance**, **Permissions**, **Telemetry**, and **Footprint**.
- 🧪 **Deterministic scoring.** Every number is reproducible from public facts.
  The model writes prose; it never touches a grade.
- 🚪 **Red-flag gates.** Remote-fetch install scripts, embedded product
  analytics, and registry **deprecations** cap the verdict — they can't be
  averaged away.
- 📦 **Real footprint.** Unpacked size + direct-dependency counts from npm & PyPI
  (estimated from the repo when unpublished).
- 🔌 **GitHub + npm + PyPI.** Paste an `owner/repo`, a URL, or `npm:pkg` /
  `pypi:pkg`.
- 🖼️ **Shareable cards** as **SVG** (zero-dependency) and **PNG**.
- 🧠 **Bring your own model** — Anthropic, OpenAI, or local **Ollama** — or run
  fully **offline** with `--no-ai`.
- 🧰 **Scriptable.** `--format json` for pipelines, `--min-score` to gate a
  dependency in CI.
- 📦 **One static binary.** No runtime, no telemetry of its own.

## Install

### `go install`

```bash
go install github.com/agenticraptor/readme-radar/cmd/readme-radar@latest
```

### Pre-built binaries

Grab a binary for your OS/arch from the
[**Releases**](https://github.com/agenticraptor/readme-radar/releases) page.

### Homebrew (macOS / Linux)

```bash
brew install agenticraptor/tap/readme-radar
```

> Available once the Homebrew tap is published — see the note in
> [`.goreleaser.yaml`](.goreleaser.yaml) to enable it.

### From source

```bash
git clone https://github.com/agenticraptor/readme-radar
cd readme-radar
make install
```

## Quickstart

```bash
# 1. Scan a GitHub repo (offline, instant, free)
readme-radar charmbracelet/bubbletea --no-ai

# 2. Vet an npm package before you install it
readme-radar npm:express

# 3. A PyPI package, as a Markdown report
readme-radar pypi:requests --format markdown

# 4. Export a shareable verdict card
readme-radar react --card react.png        # or react.svg

# 5. Machine-readable, for scripts
readme-radar npm:left-pad --no-ai --format json | jq '.score'

# 6. Gate a dependency in CI (exit non-zero below the bar)
readme-radar npm:some-new-dep --no-ai --min-score 60
```

> **Tip:** set `GITHUB_TOKEN` (a classic token with no scopes works) to raise the
> GitHub API limit from 60 to 5,000 requests/hour. Run `readme-radar doctor` to
> check your environment.

## Usage

```text
readme-radar <repo-or-package> [flags]   Scan and print the verdict card
readme-radar scan <target> [flags]       Same as above, explicitly
readme-radar card <target> [-o file]     Render a shareable SVG/PNG card
readme-radar config <init|path|show>
readme-radar doctor                      Check your environment
readme-radar version
```

Accepted targets: `owner/repo` · a `github.com` URL · `npm:<pkg>` ·
`pypi:<pkg>` · an `npmjs.com` / `pypi.org` URL.

Common flags:

| Flag | Description |
|------|-------------|
| `-f, --format` | `term` · `plain` · `markdown` · `json` (default: auto) |
| `-o, --output <file>` | Write the report to a file |
| `--card <file>` | Also write a verdict card (`.svg` or `.png`) |
| `--scale <n>` | PNG card scale factor (default `2`) |
| `--no-ai` | Skip the model; use the deterministic offline summary |
| `--provider`, `--model` | Override the LLM provider/model |
| `--token <tok>` | GitHub token (overrides `GITHUB_TOKEN`/`GH_TOKEN`) |
| `--min-score <n>` | Exit non-zero if the overall score is below `n` (for CI) |
| `--no-color` | Disable colored output |

## How it works

```
 input ─► resolve ─► GitHub + npm/PyPI facts ─► 4 deterministic analyzers
                                                       │
                                  health · telemetry · permissions · footprint
                                                       │
                                          combine ─► grade + verdict (+ gates)
                                                       │
                              narrate (model or offline) ─► render / card
```

readme-radar gathers public metadata, scores four axes **deterministically**,
then lets a model phrase the result (or falls back to an offline summary). The
grade never depends on the model. See
[**docs/how-it-works.md**](docs/how-it-works.md) for the full pipeline and
[**docs/scoring.md**](docs/scoring.md) for the exact weights and gates.

## Privacy

readme-radar reads **public** metadata (the GitHub API and the npm/PyPI
registries) and never clones or executes the analyzed project's code. It sends
nothing of its own anywhere — **unless** you enable cloud AI narration, in which
case a short, structured *facts brief* of the already-computed report is sent to
your chosen provider to write the prose. Use `--no-ai` or `--provider ollama` to
keep everything local. Full details in [SECURITY.md](SECURITY.md).

> readme-radar is a fast first screen, not a security audit. Telemetry and
> permission checks are best-effort heuristics over declared dependencies and
> manifests — a clean result reduces risk but doesn't prove safety.

## Contributing

Contributions are very welcome — see [CONTRIBUTING.md](CONTRIBUTING.md). Good
first issues include new footprint sources (Cargo, Go modules), more
analytics/telemetry SDK patterns, and richer install-script heuristics. Please
also read our [Code of Conduct](CODE_OF_CONDUCT.md).

## License

[MIT](LICENSE) © readme-radar contributors.

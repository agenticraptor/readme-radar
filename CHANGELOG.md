# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-06-16

### Added

- Initial release. 🎉
- One-command trust verdict for any GitHub repository or npm/PyPI package, from
  an `owner/repo` slug, a full URL, or an `npm:`/`pypi:` shorthand.
- Four deterministic analysis axes, each scored 0–100:
  - **Maintenance** — recency, contributors, releases, license, popularity, and
    open-issue ratio, with hard caps for archived/disabled repos.
  - **Permissions** — install scripts (preinstall/postinstall), remote-fetch
    install commands, native build steps, and process-spawning dependencies.
  - **Telemetry** — analytics/error-reporting SDKs and telemetry endpoints found
    in the dependency set and manifest.
  - **Footprint** — real unpacked install size and direct-dependency count from
    the npm/PyPI registries (estimated from the repo when unpublished).
- An overall letter grade (A–F) and verdict (trust / caution / avoid), with
  **gates** so a single critical finding (remote-fetch install script, embedded
  product analytics, or a registry **deprecation**) can't be averaged away by an
  otherwise-healthy repo.
- npm **deprecation** detection (e.g. `request`) folded into the maintenance axis.
- Plain-English narration via Anthropic, OpenAI, or local **Ollama** — with a
  deterministic **offline** summary when no model is configured. The model only
  describes facts readme-radar computed; it never scores or decides.
- Output formats: a styled terminal **verdict card**, plain text, Markdown, and
  JSON.
- Shareable **verdict cards** exported as **SVG** (zero-dependency) and **PNG**.
- `scan`, `card`, `config`, `doctor`, and `version` commands.
- `--min-score` for CI gating (non-zero exit below a threshold).
- Works unauthenticated; honors `GITHUB_TOKEN`/`GH_TOKEN` to raise the API limit.

### Security & robustness

- Untrusted text (GitHub/registry descriptions, model output) is stripped of
  ANSI escape sequences and control characters before rendering, preventing
  terminal spoofing and keeping generated SVG valid.
- Package names are validated and git refs URL-encoded, so a crafted target
  cannot manipulate a request URL.
- npm metadata is fetched from the small single-version `/latest` endpoint
  instead of the full packument, so very large packages (e.g. `@types/node`)
  scan reliably and fast.
- The analysis phase runs under an overall timeout, so a scan never hangs.
- Scanning a monorepo/workspace root surfaces a hint that the published
  package(s) may differ (e.g. scan `npm:<package>` for a specific one).

[Unreleased]: https://github.com/agenticraptor/readme-radar/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/agenticraptor/readme-radar/releases/tag/v0.1.0

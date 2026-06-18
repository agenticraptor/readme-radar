# Security Policy

## Supported versions

The latest released minor version receives security fixes. Please upgrade to the
most recent release before reporting an issue.

## Reporting a vulnerability

Please **do not** open a public issue for security problems.

Instead, use GitHub's private vulnerability reporting on this repository:
[**Report a vulnerability**](https://github.com/agenticraptor/readme-radar/security/advisories/new).
This keeps the report confidential between you and the maintainers.

Please include:

- A description of the issue and its impact.
- Steps to reproduce (a minimal proof of concept is ideal).
- Affected version(s) and platform.

We aim to acknowledge reports within **72 hours** and to provide a remediation
timeline after triage. We will credit reporters in the release notes unless you
prefer to remain anonymous.

## Scope & data handling notes

readme-radar is a local CLI that reads **public** metadata to produce a verdict.
A few things worth knowing for your own threat model:

- **What it reads:** the public GitHub REST API (repo metadata, language stats,
  contributor/release counts, and small manifest files like `package.json`) and
  the public npm / PyPI registry APIs. It does **not** clone the repo or execute
  any of the analyzed project's code.
- **What it sends:** nothing, unless you enable AI narration. When a cloud model
  (Anthropic/OpenAI) is selected, readme-radar sends a short, structured **facts
  summary** of the *already-computed* report (grades, counts, factor text) to
  that provider to write the prose. Use `--no-ai` or `--provider ollama` to keep
  everything local.
- **Credentials:** API keys are read from the environment
  (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`) and a GitHub token from
  `GITHUB_TOKEN` / `GH_TOKEN`. They are never written to the config file or
  included in error output (they travel in request headers, not URLs).
- **Untrusted text is sanitized.** Repository descriptions, package summaries,
  and model output are attacker-controllable. Before any of it is rendered,
  readme-radar strips ANSI escape sequences and control characters, so a
  malicious repo can't spoof or clobber your terminal or corrupt a generated
  SVG card.
- **Input is validated.** Package names are checked against a strict character
  set, and a git ref is URL-encoded, so a crafted target can't manipulate the
  registry/API request URL.
- **The verdict is advisory.** Telemetry and permission detection are
  **best-effort heuristics** over declared dependencies and manifests. A clean
  result is **not** a guarantee that a package is safe, and a flag is not proof
  of malice — always apply your own judgment before installing untrusted code.

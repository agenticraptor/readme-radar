# How it works

readme-radar turns a single target into a graded verdict in one pass. The
pipeline is deliberately a one-way flow: gather facts → score them
deterministically → (optionally) narrate.

```
 input ("owner/repo", URL, npm:pkg, pypi:pkg)
   │
   ▼
 source.Parse ───────────────► Target {ecosystem, owner/repo or package}
   │
   ├─ npm / pypi ──► registry ──► size · deps · install scripts · deprecation · repo URL
   │                                   │
   ▼                                   ▼
 github ──► repo metadata · languages · contributors · releases · manifests
   │
   ▼
 analyzers (pure, deterministic)
   ├─ health        maintenance score from GitHub facts
   ├─ telemetry     analytics / telemetry SDKs in the dependency set
   ├─ permissions   install scripts · native builds · process spawning
   └─ footprint     install size + direct dependency count
   │
   ▼
 score.Combine ──► overall 0–100 · grade A–F · verdict · GATES
   │
   ▼
 narrate (optional) ──► plain-English summary (model) OR deterministic offline
   │
   ▼
 render ──► terminal card · plain · Markdown · JSON      card ──► SVG · PNG
```

## Resolving the target

`source.Parse` accepts an `owner/repo` slug, a full `github.com` URL (with an
optional `/tree/<ref>`), an `npmjs.com` / `pypi.org` URL, or an explicit
`npm:` / `pypi:` / `gh:` shorthand. It does no network I/O — it only normalizes
the input into a `Target`.

## Gathering facts

For **npm/PyPI** targets, readme-radar queries the registry first. That yields
the install size, dependency list, lifecycle scripts, deprecation status, and —
crucially — the **backing GitHub repository**, so maintenance can still be
assessed. For **GitHub** targets, it reads the repo metadata directly and looks
for a `package.json` (or Python/Go manifest) to recover dependencies and
scripts, then upgrades to authoritative registry numbers when the package is
published.

GitHub calls are bounded (roughly half a dozen per scan) and use the
[Link-header pagination trick](https://docs.github.com/en/rest/guides/using-pagination-in-the-rest-api)
to count contributors and releases in a single request each. The tool works
unauthenticated; set `GITHUB_TOKEN` to raise the 60 req/hr limit to 5,000.

**Repo vs. package.** Scanning `owner/repo` analyzes that *repository's* root —
which for a monorepo (React, Babel, …) is dev tooling that may run install
scripts the published package doesn't. readme-radar detects this and prints a
hint; scan `npm:<package>` to vet the exact thing you'd install.

## Scoring

Each axis is scored independently and combined with fixed weights. Critical
findings **gate** the overall verdict so they can't be averaged away. See
[scoring.md](scoring.md) for the exact weights, thresholds, and gates.

## Narration

The narrative is the only place a model is involved, and it is **describing**,
never deciding: it receives the already-computed grades and factors and writes a
2–3 sentence summary. With no model configured (or `--no-ai`), a deterministic
offline summary is generated from the same facts. Either way the numbers are
identical.

## Output

The same `Report` drives every format: the styled terminal card, plain text,
Markdown, JSON, and the shareable SVG/PNG card. Because they all derive from one
structure, the grade you see on the card always matches the JSON.

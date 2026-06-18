# Scoring

Every number readme-radar prints is computed deterministically from observable
facts. This document is the source of truth for how the grade is produced. The
model never affects a score.

## The four axes

Each axis is scored `0–100` and converted to a letter with the same scale:

| Grade | Score |
|:-----:|:-----:|
| A | 85–100 |
| B | 70–84 |
| C | 55–69 |
| D | 40–54 |
| F | 0–39 |

### Maintenance

A weighted blend of GitHub signals (weights sum to 100):

| Signal | Weight | Notes |
|--------|:------:|-------|
| Recency of last push | 32 | Decays from "active" (≤30d) to "dormant" (>2y) |
| Contributors | 20 | Bus-factor: a single maintainer is penalized |
| Releases | 12 | Tagged releases and cadence |
| License | 10 | A recognized license vs none |
| Maturity | 10 | Age, with a small penalty for brand-new repos |
| Popularity | 8 | Stars, as a weak corroborating signal |
| Open-issue ratio | 8 | Open issues relative to stars |

**Hard caps:** an **archived** repo is capped at 25; a **disabled** repo at 15;
a forked repo loses a few points. A registry **deprecation** caps maintenance at
35 and flags it in red.

### Permissions

Starts at 100 and subtracts for capabilities that run code or touch the system:

- **Install scripts** that fetch/execute remote code (`curl … | bash` in a
  `postinstall`) — the largest penalty.
- **Install scripts** that run local code on `install` — large penalty.
- **Native build** steps (`node-gyp`) — moderate.
- **Process-spawning** dependencies (`execa`, `cross-spawn`, …) — moderate.
- **Dynamic execution** patterns found in the manifest — small.

### Telemetry

Starts at 100 (quiet) and subtracts for things that phone home:

- **Product analytics / session recording** SDKs (Segment, Mixpanel, Amplitude,
  PostHog, FullStory, …) — treated as a red flag.
- **Error / performance telemetry** (Sentry, Bugsnag, Datadog, OpenTelemetry, …)
  — a milder, "warn"-level penalty.
- **Telemetry endpoints** referenced directly in the manifest text.

Generic HTTP clients (axios, requests, …) are noted as *informational* only —
most libraries legitimately make network calls.

### Footprint

Scored from the **real unpacked install size** (npm) or artifact size (PyPI),
with a penalty as the **direct-dependency** count grows. When a package isn't
published to a registry, the size is estimated from the repository and labeled
as such.

## The overall verdict

The overall score is a weighted average of the four axes:

```
overall = 0.35·Maintenance + 0.30·Permissions + 0.20·Telemetry + 0.15·Footprint
```

…mapped to a verdict:

| Verdict | Score |
|---------|:-----:|
| **looks trustworthy** | ≥ 70 |
| **proceed with caution** | 45–69 |
| **red flags — look closely** | < 45 |

## Gates (why a "perfect" repo can still fail)

A weighted average alone lets a popular, well-maintained repo "buy back" a
serious problem with its maintenance halo. Surfacing exactly that is the point of
the tool, so certain findings **cap** the overall score:

| Finding | Caps overall at |
|---------|:---------------:|
| Registry **deprecation** | 50 (caution) |
| Install script **runs code** at install | 64 (caution) |
| Install script **fetches/executes remote code** | 44 (avoid) |
| Embedded **product analytics** | 69 (caution) |

So a pristine, 25k-star package that ships a `curl … | bash` postinstall lands
in **avoid**, not **A** — which is the whole reason to run readme-radar before
you install.

## Honesty note

Telemetry and permission detection are **best-effort heuristics** over declared
dependencies and manifests, not a sandboxed trace. A clean result reduces risk;
it does not prove safety. Treat the verdict as a fast first screen, not a
security audit.

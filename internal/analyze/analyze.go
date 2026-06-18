// Package analyze orchestrates a full scan: resolve the target, gather GitHub
// and registry facts, run the deterministic analyzers (health, telemetry,
// permissions, footprint), and combine them into a scored Report. Narration is
// added separately by the caller so the analysis stays free of LLM concerns.
package analyze

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/agenticraptor/readme-radar/internal/github"
	"github.com/agenticraptor/readme-radar/internal/health"
	"github.com/agenticraptor/readme-radar/internal/permissions"
	"github.com/agenticraptor/readme-radar/internal/registry"
	"github.com/agenticraptor/readme-radar/internal/report"
	"github.com/agenticraptor/readme-radar/internal/score"
	"github.com/agenticraptor/readme-radar/internal/telemetry"
)

// Analyzer gathers facts and assembles a Report. The two clients are
// interfaces-by-struct so tests can point them at httptest servers.
type Analyzer struct {
	GH  *github.Client
	Reg *registry.Client
	Now func() time.Time
}

// New returns an Analyzer using the public GitHub and registry endpoints.
func New() *Analyzer {
	return &Analyzer{GH: github.New(), Reg: registry.New(), Now: time.Now}
}

// Analyze runs the full pipeline for a resolved target.
func (a *Analyzer) Analyze(ctx context.Context, t report.Target) (report.Report, error) {
	now := a.Now()
	rep := report.Report{Target: t, GeneratedAt: now}
	add := func(format string, args ...any) { rep.Warnings = append(rep.Warnings, fmt.Sprintf(format, args...)) }

	// Evidence gathered for the analyzers.
	var (
		deps       []string
		scripts    map[string]string
		files      = map[string][]byte{}
		native     bool
		size       report.SizeResult
		deprecated string
	)

	// 1) Registry-first for package ecosystems; this also tells us the repo.
	if t.Ecosystem == report.EcosystemNPM || t.Ecosystem == report.EcosystemPyPI {
		reg, err := a.fetchRegistry(ctx, t)
		if err != nil {
			return rep, err // the package must exist to analyze it
		}
		deps = reg.DirectDeps
		scripts = reg.Scripts
		size = sizeFromRegistry(reg)
		deprecated = reg.Deprecated
		if reg.Owner != "" {
			t.Owner, t.Repo = reg.Owner, reg.Repo
			if t.URL == "" {
				t.URL = reg.RepoURL
			}
		} else {
			add("%s has no linked GitHub repository — maintenance health is limited", reg.Name)
		}
		rep.Target = t
	}

	// 2) GitHub facts (when we have a repo).
	if t.Owner != "" && t.Repo != "" {
		info, err := a.GH.Repo(ctx, t.Owner, t.Repo)
		if err != nil {
			if errors.Is(err, github.ErrNotFound) {
				return rep, fmt.Errorf("repository %s not found", t.Slug())
			}
			var rl *github.RateLimitError
			if errors.As(err, &rl) {
				return rep, rl
			}
			return rep, fmt.Errorf("github: %w", err)
		}
		a.enrichRepo(ctx, t, &info, add)
		rep.Repo = info

		// 3) For GitHub-origin targets, look for a package manifest to analyze.
		if t.Ecosystem == report.EcosystemGitHub {
			deps, scripts, native, size = a.fromManifests(ctx, t, info, files, add)
		}
	} else if t.Ecosystem == report.EcosystemGitHub {
		return rep, errors.New("internal: GitHub target without owner/repo")
	}

	// 4) Run the deterministic analyzers.
	rep.Health = health.Assess(rep.Repo, now)
	if rep.Repo.FullName == "" {
		// No repo at all → maintenance is unknown; keep it neutral, not punishing.
		rep.Health = report.HealthResult{Score: 60, Maintained: false, Factors: []report.Factor{{
			Name: "Maintenance", Level: report.LevelInfo, Status: report.LevelInfo.String(),
			Detail: "no GitHub repository linked — can't assess maintenance",
		}}}
	}
	if deprecated != "" {
		applyDeprecation(&rep.Health, deprecated)
		add("registry marks %s as deprecated: %s", t.Slug(), deprecated)
	}
	rep.Telemetry = telemetry.Analyze(telemetry.Input{Deps: deps, Files: files})
	rep.Permissions = permissions.Analyze(permissions.Input{
		Ecosystem: t.Ecosystem, InstallScripts: scripts, Deps: deps, Files: files, NativeBuild: native,
	})

	size.Score, size.Factors = score.Footprint(size)
	rep.Size = size

	rep.Score = score.Combine(rep.Health, rep.Telemetry, rep.Permissions, rep.Size)
	return rep, nil
}

// applyDeprecation marks a health result as deprecated: it caps the score,
// flags it as unmaintained, and prepends a red factor so it surfaces first.
func applyDeprecation(h *report.HealthResult, msg string) {
	h.Deprecated = msg
	h.Maintained = false
	if h.Score > 35 {
		h.Score = 35
	}
	h.Factors = append([]report.Factor{{
		Name: "Deprecated", Level: report.LevelBad, Status: report.LevelBad.String(),
		Detail: msg,
	}}, h.Factors...)
}

func (a *Analyzer) fetchRegistry(ctx context.Context, t report.Target) (registry.Result, error) {
	switch t.Ecosystem {
	case report.EcosystemNPM:
		r, err := a.Reg.NPM(ctx, t.Package)
		if err != nil {
			return r, fmt.Errorf("npm package %q: %w", t.Package, err)
		}
		return r, nil
	case report.EcosystemPyPI:
		r, err := a.Reg.PyPI(ctx, t.Package)
		if err != nil {
			return r, fmt.Errorf("PyPI package %q: %w", t.Package, err)
		}
		return r, nil
	}
	return registry.Result{}, nil
}

// enrichRepo adds the secondary GitHub signals, tolerating per-call failures.
func (a *Analyzer) enrichRepo(ctx context.Context, t report.Target, info *report.RepoInfo, add func(string, ...any)) {
	if langs, err := a.GH.Languages(ctx, t.Owner, t.Repo); err == nil {
		info.Languages = langs
		if info.PrimaryLang == "" && len(langs) > 0 {
			info.PrimaryLang = langs[0].Name
		}
	}
	if n, err := a.GH.CountContributors(ctx, t.Owner, t.Repo); err == nil {
		info.Contributors = n
	} else {
		add("couldn't count contributors: %v", err)
	}
	if tag, at, total, err := a.GH.LatestRelease(ctx, t.Owner, t.Repo); err == nil {
		info.LatestRelease, info.LatestTagAt, info.Releases = tag, at, total
	}
}

// fromManifests fetches package manifests from the repo, deriving dependencies,
// install scripts, native-build flags, and (for npm) an accurate size via the
// registry.
func (a *Analyzer) fromManifests(ctx context.Context, t report.Target, info report.RepoInfo, files map[string][]byte, add func(string, ...any)) (deps []string, scripts map[string]string, native bool, size report.SizeResult) {
	// npm: package.json is the richest manifest.
	if data, ok, _ := a.GH.FetchFile(ctx, t.Owner, t.Repo, t.Ref, "package.json"); ok {
		files["package.json"] = data
		pj := parsePackageJSON(data)
		deps, scripts, native = pj.deps, pj.scripts, pj.native
		if pj.monorepo {
			add("%s looks like a monorepo/workspace root — its published package(s) may differ; scan e.g. `npm:<package>` for a specific one", t.Slug())
		}
		// Prefer authoritative size/deps from the npm registry if published.
		if pj.name != "" {
			if reg, err := a.Reg.NPM(ctx, pj.name); err == nil && reg.Found {
				size = sizeFromRegistry(reg)
				if len(reg.DirectDeps) > 0 {
					deps = reg.DirectDeps
				}
				if len(reg.Scripts) > 0 {
					scripts = reg.Scripts
				}
			} else {
				size = sizeFromManifestDeps(len(deps), info)
			}
		} else {
			size = sizeFromManifestDeps(len(deps), info)
		}
		return deps, scripts, native, size
	}

	// Python manifests.
	for _, path := range []string{"pyproject.toml", "requirements.txt", "setup.py"} {
		if data, ok, _ := a.GH.FetchFile(ctx, t.Owner, t.Repo, t.Ref, path); ok {
			files[path] = data
			deps = append(deps, pythonDeps(path, data)...)
		}
	}
	if len(deps) > 0 {
		size = sizeFromManifestDeps(len(deps), info)
		return deps, nil, false, size
	}

	// Go / other: fall back to repo-size estimate.
	if data, ok, _ := a.GH.FetchFile(ctx, t.Owner, t.Repo, t.Ref, "go.mod"); ok {
		files["go.mod"] = data
		deps = goModDeps(data)
	}
	size = sizeFromRepo(info)
	return deps, nil, false, size
}

// --- size helpers ---

func sizeFromRegistry(reg registry.Result) report.SizeResult {
	src := "npm registry"
	if reg.Ecosystem == report.EcosystemPyPI {
		src = "PyPI"
	}
	return report.SizeResult{
		Known: reg.InstallBytes > 0, Source: src, Version: reg.Version,
		InstallBytes: reg.InstallBytes, DirectDeps: len(reg.DirectDeps),
	}
}

func sizeFromManifestDeps(nDeps int, info report.RepoInfo) report.SizeResult {
	return report.SizeResult{Known: false, Source: "repo manifest", DirectDeps: nDeps,
		InstallBytes: int64(info.SizeKB) * 1024}
}

func sizeFromRepo(info report.RepoInfo) report.SizeResult {
	return report.SizeResult{Known: false, Source: "repo", InstallBytes: int64(info.SizeKB) * 1024}
}

// --- manifest parsing ---

type packageJSON struct {
	name     string
	deps     []string
	scripts  map[string]string
	native   bool
	monorepo bool // private root or workspaces present → published package differs
}

func parsePackageJSON(data []byte) packageJSON {
	var raw struct {
		Name         string            `json:"name"`
		Private      bool              `json:"private"`
		Workspaces   json.RawMessage   `json:"workspaces"`
		Dependencies map[string]string `json:"dependencies"`
		Scripts      map[string]string `json:"scripts"`
		GypFile      json.RawMessage   `json:"gypfile"`
	}
	if json.Unmarshal(data, &raw) != nil {
		return packageJSON{}
	}
	pj := packageJSON{name: raw.Name}
	for k := range raw.Dependencies {
		pj.deps = append(pj.deps, k)
	}
	pj.scripts = lifecycle(raw.Scripts)
	if g := strings.TrimSpace(string(raw.GypFile)); g == "true" {
		pj.native = true
	}
	if _, ok := raw.Dependencies["node-gyp"]; ok {
		pj.native = true
	}
	if raw.Private || len(raw.Workspaces) > 0 || raw.Name == "" {
		pj.monorepo = true
	}
	return pj
}

func lifecycle(s map[string]string) map[string]string {
	if s == nil {
		return nil
	}
	out := map[string]string{}
	for _, k := range []string{"preinstall", "install", "postinstall"} {
		if v, ok := s[k]; ok && strings.TrimSpace(v) != "" {
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func pythonDeps(path string, data []byte) []string {
	var out []string
	switch {
	case strings.HasSuffix(path, "requirements.txt"):
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
				continue
			}
			out = append(out, depName(line))
		}
	case strings.HasSuffix(path, "pyproject.toml"):
		// naive: collect quoted requirement strings under [dependencies]/deps
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "\"") && strings.Contains(line, "\"") {
				inner := strings.Trim(line, "\", ")
				if n := depName(inner); n != "" && !strings.ContainsAny(n, "[]{}=") {
					out = append(out, n)
				}
			}
		}
	}
	return dedupe(out)
}

func goModDeps(data []byte) []string {
	var out []string
	in := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "require ("):
			in = true
		case in && line == ")":
			in = false
		case in && line != "":
			out = append(out, strings.Fields(line)[0])
		case strings.HasPrefix(line, "require ") && !strings.Contains(line, "("):
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				out = append(out, fields[1])
			}
		}
	}
	return dedupe(out)
}

func depName(s string) string {
	s = strings.TrimSpace(s)
	cut := strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == '=' || r == '>' || r == '<' || r == '!' || r == '~' || r == ';' || r == '[' || r == '(' || r == ','
	})
	if len(cut) > 0 {
		return cut[0]
	}
	return ""
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

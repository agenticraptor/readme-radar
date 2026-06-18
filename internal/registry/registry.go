// Package registry fetches install-footprint and dependency facts from the npm
// and PyPI registries, and derives the backing GitHub repository when the
// package declares one. Both registries expose simple public JSON APIs.
package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/agenticraptor/readme-radar/internal/report"
	"github.com/agenticraptor/readme-radar/internal/textutil"
)

// Result is the normalized view of a package across ecosystems.
type Result struct {
	Found        bool
	Ecosystem    report.Ecosystem
	Name         string
	Version      string
	InstallBytes int64
	FileCount    int
	DirectDeps   []string
	Scripts      map[string]string // npm lifecycle scripts (preinstall/install/postinstall/...)
	RepoURL      string
	Owner        string
	Repo         string
	Description  string
	License      string
	Deprecated   string // non-empty if the registry marks this version deprecated
}

// Client talks to registry.npmjs.org and pypi.org.
type Client struct {
	NPMBase  string
	PyPIBase string
	HTTP     *http.Client
}

// New returns a Client pointed at the public registries.
func New() *Client {
	return &Client{
		NPMBase:  "https://registry.npmjs.org",
		PyPIBase: "https://pypi.org",
		HTTP:     &http.Client{Timeout: 20 * time.Second},
	}
}

// NPM fetches metadata for an npm package. It requests the single-version
// "/latest" manifest rather than the full packument: the packument for popular
// packages can be tens of megabytes (every version ever published), while the
// latest manifest is a few kilobytes and carries everything we need — scripts,
// dependencies, unpacked size, repository, and deprecation status.
func (c *Client) NPM(ctx context.Context, name string) (Result, error) {
	var m npmManifest
	if err := c.getJSON(ctx, c.NPMBase+"/"+npmPath(name)+"/latest", &m); err != nil {
		return Result{}, err
	}
	res := Result{
		Found:        true,
		Ecosystem:    report.EcosystemNPM,
		Name:         name,
		Version:      m.Version,
		Description:  textutil.Safe(m.Description),
		InstallBytes: m.Dist.UnpackedSize,
		FileCount:    m.Dist.FileCount,
		DirectDeps:   sortedKeys(m.Dependencies),
		Scripts:      lifecycleScripts(m.Scripts),
		License:      normalizeLicense(m.License),
		Deprecated:   deprecatedMsg(m.Deprecated),
	}
	res.RepoURL, res.Owner, res.Repo = repoFrom(m.Repository.URL)
	return res, nil
}

// PyPI fetches metadata for a PyPI project.
func (c *Client) PyPI(ctx context.Context, name string) (Result, error) {
	var doc pypiDoc
	if err := c.getJSON(ctx, fmt.Sprintf("%s/pypi/%s/json", c.PyPIBase, name), &doc); err != nil {
		return Result{}, err
	}
	res := Result{Found: true, Ecosystem: report.EcosystemPyPI, Name: name,
		Version:     doc.Info.Version,
		Description: textutil.Safe(doc.Info.Summary),
		License:     textutil.Safe(doc.Info.License),
	}
	res.InstallBytes, res.FileCount = pickArtifact(doc.URLs)
	res.DirectDeps = runtimeRequires(doc.Info.RequiresDist)
	res.RepoURL, res.Owner, res.Repo = repoFromMap(doc.Info.ProjectURLs, doc.Info.HomePage)
	return res, nil
}

// --- npm payloads ---

// npmManifest is the single-version document returned by ".../<pkg>/latest".
type npmManifest struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Dependencies map[string]string `json:"dependencies"`
	Scripts      map[string]string `json:"scripts"`
	Repository   repoRef           `json:"repository"`
	License      json.RawMessage   `json:"license"`
	Deprecated   json.RawMessage   `json:"deprecated"`
	Dist         struct {
		UnpackedSize int64 `json:"unpackedSize"`
		FileCount    int   `json:"fileCount"`
	} `json:"dist"`
}

// repoRef tolerates both {"url":...} objects and bare string repository fields.
type repoRef struct {
	URL string
}

func (r *repoRef) UnmarshalJSON(b []byte) error {
	b = []byte(strings.TrimSpace(string(b)))
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		r.URL = s
		return nil
	}
	var obj struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(b, &obj); err != nil {
		return nil // be tolerant: ignore unusual shapes
	}
	r.URL = obj.URL
	return nil
}

// --- pypi payloads ---

type pypiDoc struct {
	Info struct {
		Version      string            `json:"version"`
		Summary      string            `json:"summary"`
		License      string            `json:"license"`
		HomePage     string            `json:"home_page"`
		ProjectURLs  map[string]string `json:"project_urls"`
		RequiresDist []string          `json:"requires_dist"`
	} `json:"info"`
	URLs []struct {
		PackageType string `json:"packagetype"`
		Size        int64  `json:"size"`
	} `json:"urls"`
}

// --- helpers ---

func (c *Client) getJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "readme-radar")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("package not found in registry")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("registry: %s", resp.Status)
	}
	return json.Unmarshal(body, out)
}

func npmPath(name string) string {
	// scoped packages keep their slash but the "@" is left intact
	return strings.ReplaceAll(name, "/", "%2F")
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// lifecycleScripts keeps only the install-time lifecycle hooks (the ones that
// run arbitrary code on `npm install`).
func lifecycleScripts(s map[string]string) map[string]string {
	if s == nil {
		return nil
	}
	out := map[string]string{}
	for _, k := range []string{"preinstall", "install", "postinstall", "preuninstall", "postuninstall", "prepare"} {
		if v, ok := s[k]; ok && strings.TrimSpace(v) != "" {
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// deprecatedMsg normalizes npm's "deprecated" field, which may be a message
// string or a boolean, into a human-readable reason (empty = not deprecated).
func deprecatedMsg(raw json.RawMessage) string {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" || s == "false" {
		return ""
	}
	if s == "true" {
		return "this version is deprecated"
	}
	var msg string
	if json.Unmarshal(raw, &msg) == nil && strings.TrimSpace(msg) != "" {
		return strings.TrimSpace(msg)
	}
	return "this version is deprecated"
}

var extraMarker = regexp.MustCompile(`;\s*extra\s*==`)

// runtimeRequires returns the names of unconditional (non-extra) dependencies.
func runtimeRequires(reqs []string) []string {
	var out []string
	for _, r := range reqs {
		if extraMarker.MatchString(r) {
			continue // optional, only with extras
		}
		name := strings.FieldsFunc(r, func(c rune) bool {
			return c == ' ' || c == '(' || c == '>' || c == '<' || c == '=' || c == '!' || c == '~' || c == ';' || c == '['
		})
		if len(name) > 0 && name[0] != "" {
			out = append(out, name[0])
		}
	}
	sort.Strings(out)
	return out
}

func pickArtifact(urls []struct {
	PackageType string `json:"packagetype"`
	Size        int64  `json:"size"`
}) (int64, int) {
	var wheel, sdist int64
	count := len(urls)
	for _, u := range urls {
		switch u.PackageType {
		case "bdist_wheel":
			if u.Size > wheel {
				wheel = u.Size
			}
		case "sdist":
			if u.Size > sdist {
				sdist = u.Size
			}
		}
	}
	if wheel > 0 {
		return wheel, count
	}
	return sdist, count
}

func normalizeLicense(raws ...json.RawMessage) string {
	for _, raw := range raws {
		if len(raw) == 0 {
			continue
		}
		s := strings.TrimSpace(string(raw))
		if s == "" || s == "null" {
			continue
		}
		if s[0] == '"' {
			var str string
			if json.Unmarshal(raw, &str) == nil && str != "" {
				return str
			}
		}
		var obj struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(raw, &obj) == nil && obj.Type != "" {
			return obj.Type
		}
	}
	return ""
}

var ghRe = regexp.MustCompile(`github\.com[/:]+([A-Za-z0-9_.-]+)/([A-Za-z0-9_.-]+?)(?:\.git)?(?:[/#].*)?$`)

func repoFrom(urls ...string) (repoURL, owner, repo string) {
	for _, u := range urls {
		if u == "" {
			continue
		}
		if m := ghRe.FindStringSubmatch(u); m != nil {
			owner, repo = m[1], strings.TrimSuffix(m[2], ".git")
			return "https://github.com/" + owner + "/" + repo, owner, repo
		}
	}
	return "", "", ""
}

func repoFromMap(m map[string]string, extra ...string) (string, string, string) {
	// Prefer keys that clearly point at source.
	order := []string{"Source", "Source Code", "Repository", "Code", "GitHub", "Homepage", "Home"}
	for _, k := range order {
		if v, ok := m[k]; ok {
			if u, o, r := repoFrom(v); u != "" {
				return u, o, r
			}
		}
	}
	for _, v := range m {
		if u, o, r := repoFrom(v); u != "" {
			return u, o, r
		}
	}
	return repoFrom(extra...)
}

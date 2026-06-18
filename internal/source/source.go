// Package source turns whatever the user typed — a GitHub URL, an "owner/repo"
// slug, an npm/PyPI URL, or an "npm:pkg"/"pypi:pkg" shorthand — into a resolved
// report.Target. It performs no network I/O; enrichment happens later.
package source

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/agenticraptor/readme-radar/internal/report"
)

// ErrEmpty is returned when no target was supplied.
var ErrEmpty = errors.New("no repository or package specified")

var (
	// owner/repo segments: letters, digits, dot, dash, underscore.
	segRe  = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
	refCut = map[string]bool{"tree": true, "blob": true, "commit": true, "releases": true, "tags": true}
)

// Parse resolves a raw user string into a Target.
func Parse(raw string) (report.Target, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return report.Target{}, ErrEmpty
	}

	// Explicit ecosystem shorthands: "npm:foo", "pypi:bar", "gh:owner/repo".
	if eco, rest, ok := splitScheme(s); ok {
		switch eco {
		case "npm":
			return npmTarget(raw, rest)
		case "pypi", "pip":
			return pypiTarget(raw, rest)
		case "gh", "github":
			return githubTarget(raw, rest)
		case "http", "https":
			return fromURL(raw, rest)
		}
	}

	// Bare "owner/repo" with no host is treated as GitHub.
	if !strings.Contains(s, "://") && strings.Count(s, "/") >= 1 && !strings.Contains(s, ".") {
		return githubTarget(raw, s)
	}
	// Anything containing a host falls through to URL handling.
	if strings.Contains(s, "github.com") || strings.Contains(s, "npmjs.com") || strings.Contains(s, "pypi.org") {
		return fromURL(raw, stripScheme(s))
	}
	// Last resort: "owner/repo" shaped → GitHub.
	if strings.Count(s, "/") >= 1 {
		return githubTarget(raw, s)
	}
	return report.Target{}, fmt.Errorf("could not understand %q — try owner/repo, a GitHub URL, or npm:<pkg>", raw)
}

func splitScheme(s string) (scheme, rest string, ok bool) {
	if i := strings.Index(s, "://"); i > 0 {
		return strings.ToLower(s[:i]), s[i+3:], true
	}
	if i := strings.Index(s, ":"); i > 0 {
		scheme = strings.ToLower(s[:i])
		switch scheme {
		case "npm", "pypi", "pip", "gh", "github":
			return scheme, s[i+1:], true
		}
	}
	return "", "", false
}

func stripScheme(s string) string {
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	return strings.TrimPrefix(s, "www.")
}

func fromURL(raw, hostPath string) (report.Target, error) {
	hostPath = strings.TrimPrefix(hostPath, "www.")
	host, path, _ := strings.Cut(hostPath, "/")
	host = strings.ToLower(host)
	switch {
	case strings.HasSuffix(host, "github.com"):
		return githubTarget(raw, path)
	case strings.HasSuffix(host, "npmjs.com"):
		// /package/<name> or /package/@scope/name
		path = strings.TrimPrefix(path, "package/")
		return npmTarget(raw, path)
	case strings.HasSuffix(host, "pypi.org"):
		// /project/<name>/
		path = strings.TrimPrefix(path, "project/")
		return pypiTarget(raw, path)
	}
	// Unknown host but looks like a path → try GitHub semantics.
	if path != "" {
		return githubTarget(raw, path)
	}
	return report.Target{}, fmt.Errorf("unsupported URL host %q", host)
}

func githubTarget(raw, path string) (report.Target, error) {
	path = strings.TrimPrefix(strings.TrimSpace(path), "/")
	path = strings.TrimSuffix(path, "/")
	path = strings.TrimSuffix(path, ".git")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return report.Target{}, fmt.Errorf("expected owner/repo, got %q", path)
	}
	owner, repo := parts[0], parts[1]
	if !segRe.MatchString(owner) || !segRe.MatchString(repo) {
		return report.Target{}, fmt.Errorf("invalid owner/repo in %q", path)
	}
	t := report.Target{
		Raw:       raw,
		Ecosystem: report.EcosystemGitHub,
		Owner:     owner,
		Repo:      repo,
		URL:       "https://github.com/" + owner + "/" + repo,
	}
	// Optional ref: .../tree/<ref> or .../commit/<sha>.
	if len(parts) >= 4 && refCut[parts[2]] {
		t.Ref = parts[3]
	}
	return t, nil
}

// pkgNameRe permits an optional npm @scope plus a conservative character set,
// rejecting anything that could manipulate a registry request URL (slashes
// beyond the scope, "..", control characters, query/fragment markers).
var pkgNameRe = regexp.MustCompile(`^(@[A-Za-z0-9][A-Za-z0-9._-]*/)?[A-Za-z0-9][A-Za-z0-9._~-]*$`)

func validatePkg(kind, name string) error {
	if name == "" {
		return fmt.Errorf("empty %s package name", kind)
	}
	if len(name) > 214 || strings.Contains(name, "..") || !pkgNameRe.MatchString(name) {
		return fmt.Errorf("invalid %s package name %q", kind, name)
	}
	return nil
}

func npmTarget(raw, name string) (report.Target, error) {
	name = cleanPkg(name)
	if err := validatePkg("npm", name); err != nil {
		return report.Target{}, err
	}
	return report.Target{
		Raw:       raw,
		Ecosystem: report.EcosystemNPM,
		Package:   name,
		URL:       "https://www.npmjs.com/package/" + name,
	}, nil
}

func pypiTarget(raw, name string) (report.Target, error) {
	name = cleanPkg(name)
	if err := validatePkg("PyPI", name); err != nil {
		return report.Target{}, err
	}
	return report.Target{
		Raw:       raw,
		Ecosystem: report.EcosystemPyPI,
		Package:   name,
		URL:       "https://pypi.org/project/" + name + "/",
	}, nil
}

// cleanPkg trims a package path down to its name, preserving an @scope.
func cleanPkg(s string) string {
	s = strings.Trim(strings.TrimSpace(s), "/")
	if s == "" {
		return ""
	}
	if strings.HasPrefix(s, "@") {
		// scoped: keep first two segments (@scope/name)
		parts := strings.SplitN(s, "/", 3)
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
		return s
	}
	// unscoped: first segment only
	return strings.SplitN(s, "/", 2)[0]
}

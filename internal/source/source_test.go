package source

import (
	"testing"

	"github.com/agenticraptor/readme-radar/internal/report"
)

func TestParseGitHub(t *testing.T) {
	cases := []struct {
		in    string
		owner string
		repo  string
		ref   string
	}{
		{"https://github.com/charmbracelet/bubbletea", "charmbracelet", "bubbletea", ""},
		{"github.com/charmbracelet/bubbletea", "charmbracelet", "bubbletea", ""},
		{"charmbracelet/bubbletea", "charmbracelet", "bubbletea", ""},
		{"https://github.com/charmbracelet/bubbletea.git", "charmbracelet", "bubbletea", ""},
		{"https://github.com/cli/cli/tree/v2.0.0", "cli", "cli", "v2.0.0"},
		{"https://github.com/cli/cli/blob/main/README.md", "cli", "cli", "main"},
		{"gh:owner/repo", "owner", "repo", ""},
		{"https://www.github.com/owner/repo/", "owner", "repo", ""},
	}
	for _, c := range cases {
		got, err := Parse(c.in)
		if err != nil {
			t.Fatalf("Parse(%q) error: %v", c.in, err)
		}
		if got.Ecosystem != report.EcosystemGitHub {
			t.Errorf("Parse(%q) ecosystem = %q, want github", c.in, got.Ecosystem)
		}
		if got.Owner != c.owner || got.Repo != c.repo {
			t.Errorf("Parse(%q) = %s/%s, want %s/%s", c.in, got.Owner, got.Repo, c.owner, c.repo)
		}
		if got.Ref != c.ref {
			t.Errorf("Parse(%q) ref = %q, want %q", c.in, got.Ref, c.ref)
		}
	}
}

func TestParseNPM(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"npm:left-pad", "left-pad"},
		{"npm:@types/node", "@types/node"},
		{"https://www.npmjs.com/package/express", "express"},
		{"https://www.npmjs.com/package/@vue/cli", "@vue/cli"},
	}
	for _, c := range cases {
		got, err := Parse(c.in)
		if err != nil {
			t.Fatalf("Parse(%q) error: %v", c.in, err)
		}
		if got.Ecosystem != report.EcosystemNPM || got.Package != c.want {
			t.Errorf("Parse(%q) = %q (%s), want npm %q", c.in, got.Package, got.Ecosystem, c.want)
		}
	}
}

func TestParsePyPI(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"pypi:requests", "requests"},
		{"pip:flask", "flask"},
		{"https://pypi.org/project/numpy/", "numpy"},
	}
	for _, c := range cases {
		got, err := Parse(c.in)
		if err != nil {
			t.Fatalf("Parse(%q) error: %v", c.in, err)
		}
		if got.Ecosystem != report.EcosystemPyPI || got.Package != c.want {
			t.Errorf("Parse(%q) = %q (%s), want pypi %q", c.in, got.Package, got.Ecosystem, c.want)
		}
	}
}

func TestParseErrors(t *testing.T) {
	for _, in := range []string{"", "   ", "justaword", "owner/"} {
		if _, err := Parse(in); err == nil {
			t.Errorf("Parse(%q) expected error, got nil", in)
		}
	}
}

func TestRejectsMaliciousPackageNames(t *testing.T) {
	// Names that could manipulate a registry request URL must be rejected.
	for _, in := range []string{
		"npm:../../../etc/passwd",
		"npm:foo bar",
		"npm:foo@1.2.3",
		"pypi:..",
		"npm:a\x1b[31m",
		"npm:.hidden",
	} {
		if _, err := Parse(in); err == nil {
			t.Errorf("Parse(%q) should be rejected", in)
		}
	}
}

func TestAcceptsValidPackageNames(t *testing.T) {
	for _, in := range []string{"npm:left-pad", "npm:@vue/cli", "npm:lodash.merge", "pypi:requests", "pypi:scikit-learn"} {
		if _, err := Parse(in); err != nil {
			t.Errorf("Parse(%q) should be accepted, got %v", in, err)
		}
	}
}

func TestSlug(t *testing.T) {
	g, _ := Parse("owner/repo")
	if g.Slug() != "owner/repo" {
		t.Errorf("Slug = %q", g.Slug())
	}
	n, _ := Parse("npm:express")
	if n.Slug() != "express" {
		t.Errorf("Slug = %q", n.Slug())
	}
}

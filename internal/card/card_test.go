package card

import (
	"bytes"
	"image/png"
	"strings"
	"testing"

	"github.com/agenticraptor/readme-radar/internal/report"
)

func sample() report.Report {
	return report.Report{
		Target: report.Target{Ecosystem: report.EcosystemGitHub, Owner: "charm", Repo: "bubbletea"},
		Repo:   report.RepoInfo{FullName: "charm/bubbletea", Stars: 28900, License: "MIT"},
		Size:   report.SizeResult{Known: true, InstallBytes: 1258291, DirectDeps: 7},
		Score: report.ScoreResult{Overall: 86, Grade: report.GradeA, Verdict: report.VerdictTrust, Headline: "Healthy and lean.",
			Axes: []report.Axis{{Name: "Maintenance", Score: 92, Grade: report.GradeA}, {Name: "Footprint", Score: 70, Grade: report.GradeB}}},
		Narrative: report.Narrative{Summary: "bubbletea is a delightful TUI framework that is actively maintained and very widely used.", Source: "offline"},
	}
}

func TestSVGWellFormed(t *testing.T) {
	out := SVG(sample())
	if !strings.HasPrefix(out, "<svg") || !strings.HasSuffix(out, "</svg>") {
		t.Fatalf("SVG not well-formed at edges")
	}
	for _, want := range []string{"readme-radar", "Maintenance", "TRUST", "MIT"} {
		if !strings.Contains(out, want) {
			t.Errorf("SVG missing %q", want)
		}
	}
	// balanced tags (rough check: no stray unescaped ampersand)
	if strings.Contains(out, " & ") {
		t.Errorf("SVG contains unescaped ampersand")
	}
}

func TestPNGEncodes(t *testing.T) {
	var buf bytes.Buffer
	if err := PNG(sample(), &buf, 2); err != nil {
		t.Fatalf("PNG: %v", err)
	}
	img, err := png.Decode(&buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if img.Bounds().Dx() != width*2 || img.Bounds().Dy() != height*2 {
		t.Errorf("unexpected dims: %v", img.Bounds())
	}
}

func TestWrap(t *testing.T) {
	lines := wrap("the quick brown fox jumps over the lazy dog again and again", 20, 2)
	if len(lines) != 2 {
		t.Fatalf("want 2 lines, got %d: %v", len(lines), lines)
	}
	for _, l := range lines {
		if len([]rune(l)) > 20 {
			t.Errorf("line too long (%d runes): %q", len([]rune(l)), l)
		}
	}
}

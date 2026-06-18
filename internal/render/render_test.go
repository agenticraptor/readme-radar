package render

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/agenticraptor/readme-radar/internal/report"
)

func sample() report.Report {
	return report.Report{
		Target:      report.Target{Ecosystem: report.EcosystemGitHub, Owner: "charm", Repo: "bubbletea", URL: "https://github.com/charm/bubbletea"},
		Repo:        report.RepoInfo{FullName: "charm/bubbletea", Description: "A TUI framework", Stars: 28900, Contributors: 120, Releases: 40, License: "MIT", PrimaryLang: "Go", PushedAt: time.Now().AddDate(0, 0, -3)},
		Health:      report.HealthResult{Score: 92, Factors: []report.Factor{{Name: "Last commit", Level: report.LevelGood, Status: "good", Detail: "active"}}},
		Telemetry:   report.TelemetryResult{Score: 95, Level: report.LevelGood, Status: "good", Factors: []report.Factor{{Name: "Telemetry", Level: report.LevelGood, Status: "good", Detail: "none"}}},
		Permissions: report.PermissionsResult{Score: 88, Capabilities: []report.Factor{{Name: "Install scripts", Level: report.LevelGood, Status: "good", Detail: "none"}}},
		Size:        report.SizeResult{Known: true, Source: "repo", InstallBytes: 1258291, DirectDeps: 7, Score: 70, Factors: []report.Factor{{Name: "Install size", Level: report.LevelWarn, Status: "warn", Detail: "1.2 MB"}}},
		Score: report.ScoreResult{Overall: 86, Grade: report.GradeA, Verdict: report.VerdictTrust, Headline: "Healthy and lean.",
			Axes: []report.Axis{{Name: "Maintenance", Score: 92, Grade: report.GradeA}, {Name: "Permissions", Score: 88, Grade: report.GradeA}, {Name: "Telemetry", Score: 95, Grade: report.GradeA}, {Name: "Footprint", Score: 70, Grade: report.GradeB}}},
		Narrative:   report.Narrative{Summary: "bubbletea is a TUI framework. Safe to adopt.", Bullets: []string{"Active", "Lean"}, Source: "offline"},
		GeneratedAt: time.Now(),
	}
}

func TestRenderAllFormats(t *testing.T) {
	for _, f := range []Format{FormatTerm, FormatPlain, FormatMarkdown, FormatJSON} {
		out, err := Render(sample(), Options{Format: f, NoColor: true})
		if err != nil {
			t.Fatalf("format %s: %v", f, err)
		}
		if strings.TrimSpace(out) == "" {
			t.Errorf("format %s produced empty output", f)
		}
	}
}

func TestJSONIsValid(t *testing.T) {
	out, _ := Render(sample(), Options{Format: FormatJSON})
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := m["score"]; !ok {
		t.Errorf("missing score in JSON")
	}
}

func TestMarkdownHasGradeAndAxes(t *testing.T) {
	out, _ := Render(sample(), Options{Format: FormatMarkdown})
	if !strings.Contains(out, "grade A") || !strings.Contains(out, "Maintenance") {
		t.Errorf("markdown missing expected content:\n%s", out)
	}
}

func TestPlainHasNoANSI(t *testing.T) {
	out, _ := Render(sample(), Options{Format: FormatPlain})
	if strings.Contains(out, "\x1b[") {
		t.Errorf("plain output contains ANSI escapes")
	}
}

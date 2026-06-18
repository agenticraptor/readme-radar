package narrate

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/agenticraptor/readme-radar/internal/llm"
	"github.com/agenticraptor/readme-radar/internal/report"
)

func sampleReport() report.Report {
	return report.Report{
		Target: report.Target{Ecosystem: report.EcosystemNPM, Package: "leftpad", URL: "https://npmjs.com/package/leftpad"},
		Repo:   report.RepoInfo{FullName: "stevemao/left-pad", Description: "String left pad", Stars: 1000, License: "MIT"},
		Health: report.HealthResult{Score: 80, Factors: []report.Factor{{Name: "Last commit", Detail: "active"}}},
		Size:   report.SizeResult{Known: true, InstallBytes: 4096, DirectDeps: 0},
		Score: report.ScoreResult{Overall: 82, Grade: report.GradeB, Verdict: report.VerdictTrust,
			Headline: "Healthy and lean.", Axes: []report.Axis{{Name: "Maintenance", Grade: report.GradeB}}},
	}
}

func TestOfflineNarrative(t *testing.T) {
	n := Narrate(context.Background(), sampleReport(), nil)
	if n.Source != "offline" {
		t.Errorf("source = %q, want offline", n.Source)
	}
	if !strings.Contains(n.Summary, "left-pad") && !strings.Contains(n.Summary, "leftpad") {
		t.Errorf("summary missing name: %q", n.Summary)
	}
	if n.Summary == "" {
		t.Errorf("empty summary")
	}
}

type fakeClient struct {
	resp string
	err  error
}

func (f fakeClient) Name() string  { return "fake" }
func (f fakeClient) Model() string { return "fake-1" }
func (f fakeClient) Complete(context.Context, llm.Request) (string, error) {
	return f.resp, f.err
}

func TestModelNarrative(t *testing.T) {
	c := fakeClient{resp: "Here you go:\n```json\n{\"summary\":\"It pads strings. Safe to use.\",\"bullets\":[\"tiny\",\"no deps\"]}\n```"}
	n := Narrate(context.Background(), sampleReport(), c)
	if n.Source != "fake" {
		t.Errorf("source = %q", n.Source)
	}
	if !strings.HasPrefix(n.Summary, "It pads strings") || len(n.Bullets) != 2 {
		t.Errorf("parse failed: %+v", n)
	}
}

func TestModelErrorFallsBack(t *testing.T) {
	c := fakeClient{err: errors.New("boom")}
	n := Narrate(context.Background(), sampleReport(), c)
	if !strings.HasPrefix(n.Source, "offline") {
		t.Errorf("expected offline fallback, got %q", n.Source)
	}
}

func TestParseGarbageFallsBack(t *testing.T) {
	c := fakeClient{resp: "no json here"}
	n := Narrate(context.Background(), sampleReport(), c)
	if n.Source != "offline" {
		t.Errorf("garbage response should fall back to offline, got %q", n.Source)
	}
}

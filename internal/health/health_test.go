package health

import (
	"testing"
	"time"

	"github.com/agenticraptor/readme-radar/internal/report"
)

func TestHealthyRepoScoresHigh(t *testing.T) {
	now := time.Date(2026, 6, 16, 0, 0, 0, 0, time.UTC)
	info := report.RepoInfo{
		Stars:        12000,
		Contributors: 80,
		Releases:     40,
		License:      "MIT",
		CreatedAt:    now.AddDate(-4, 0, 0),
		PushedAt:     now.AddDate(0, 0, -3),
		OpenIssues:   120,
	}
	res := Assess(info, now)
	if res.Score < 85 {
		t.Errorf("expected A-grade health, got %d", res.Score)
	}
	if !res.Maintained {
		t.Errorf("expected maintained=true")
	}
}

func TestArchivedRepoIsCapped(t *testing.T) {
	now := time.Now()
	info := report.RepoInfo{
		Stars: 9000, Contributors: 50, Releases: 30, License: "MIT",
		CreatedAt: now.AddDate(-5, 0, 0), PushedAt: now.AddDate(0, 0, -2),
		Archived: true,
	}
	res := Assess(info, now)
	if res.Score > 25 {
		t.Errorf("archived repo should be capped <=25, got %d", res.Score)
	}
	if res.Maintained {
		t.Errorf("archived repo should not be 'maintained'")
	}
}

func TestDormantSoloRepoScoresLow(t *testing.T) {
	now := time.Now()
	info := report.RepoInfo{
		Stars: 4, Contributors: 1, Releases: 0, License: "none",
		CreatedAt: now.AddDate(-3, 0, 0), PushedAt: now.AddDate(-2, -6, 0),
	}
	res := Assess(info, now)
	if res.Score > 45 {
		t.Errorf("dormant solo repo should score low, got %d", res.Score)
	}
}

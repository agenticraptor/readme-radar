package telemetry

import (
	"testing"

	"github.com/agenticraptor/readme-radar/internal/report"
)

func TestCleanPackage(t *testing.T) {
	res := Analyze(Input{Deps: []string{"lodash", "chalk", "commander"}})
	if res.Level != report.LevelGood || res.Score < 85 {
		t.Errorf("expected clean/good, got level=%v score=%d", res.Level, res.Score)
	}
}

func TestAnalyticsHeavy(t *testing.T) {
	res := Analyze(Input{Deps: []string{"mixpanel", "@segment/analytics-node", "axios"}})
	if res.Level != report.LevelBad {
		t.Errorf("expected bad level for analytics-heavy pkg, got %v (score %d)", res.Level, res.Score)
	}
	if len(res.AnalyticsLibs) < 2 {
		t.Errorf("expected analytics libs detected, got %v", res.AnalyticsLibs)
	}
}

func TestErrorTelemetryIsMilder(t *testing.T) {
	res := Analyze(Input{Deps: []string{"@sentry/node", "express"}})
	if res.Level != report.LevelWarn {
		t.Errorf("error telemetry should warn (not good/bad), got %v", res.Level)
	}
	if res.Score < 80 {
		t.Errorf("single error-telemetry dep should be a mild penalty, got %d", res.Score)
	}
}

func TestTextScanCatchesEndpoints(t *testing.T) {
	files := map[string][]byte{"index.js": []byte(`fetch("https://api.mixpanel.com/track")`)}
	res := Analyze(Input{Deps: []string{"node-fetch"}, Files: files})
	if len(res.AnalyticsLibs) == 0 {
		t.Errorf("expected mixpanel.com endpoint to be detected")
	}
}

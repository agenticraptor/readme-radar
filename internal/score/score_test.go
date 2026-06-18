package score

import (
	"testing"

	"github.com/agenticraptor/readme-radar/internal/report"
)

func TestFootprintLeanVsHeavy(t *testing.T) {
	lean, _ := Footprint(report.SizeResult{Known: true, InstallBytes: 80 << 10, DirectDeps: 0})
	heavy, _ := Footprint(report.SizeResult{Known: true, InstallBytes: 45 << 20, DirectDeps: 60})
	if lean <= heavy {
		t.Errorf("lean (%d) should beat heavy (%d)", lean, heavy)
	}
	if lean < 85 {
		t.Errorf("tiny zero-dep package should score high, got %d", lean)
	}
}

func TestUnknownSizeIsNeutral(t *testing.T) {
	s, fs := Footprint(report.SizeResult{Known: false})
	if s != 70 || len(fs) == 0 {
		t.Errorf("unknown size should be neutral 70 with a factor, got %d", s)
	}
}

func TestCombineWeighting(t *testing.T) {
	good := report.HealthResult{Score: 95}
	cleanPerm := report.PermissionsResult{Score: 100}
	quiet := report.TelemetryResult{Score: 100}
	size := report.SizeResult{Score: 95}
	res := Combine(good, quiet, cleanPerm, size)
	if res.Grade != report.GradeA || res.Verdict != report.VerdictTrust {
		t.Errorf("all-good should be grade A/trust, got %s/%s", res.Grade, res.Verdict)
	}

	// A perfectly lean, well-maintained package that runs install scripts and
	// ships analytics should be dragged down out of "trust".
	risky := Combine(good, report.TelemetryResult{Score: 40}, report.PermissionsResult{Score: 35}, report.SizeResult{Score: 100})
	if risky.Verdict == report.VerdictTrust {
		t.Errorf("risky perms+telemetry should not be 'trust', got %s (%d)", risky.Verdict, risky.Overall)
	}
}

func TestDeprecationGate(t *testing.T) {
	// Otherwise-pristine, popular package but registry-deprecated.
	dep := Combine(
		report.HealthResult{Score: 35, Deprecated: "use something else"},
		report.TelemetryResult{Score: 100},
		report.PermissionsResult{Score: 100},
		report.SizeResult{Score: 90},
	)
	if dep.Verdict == report.VerdictTrust {
		t.Errorf("deprecated package must not be 'trust', got %s (%d)", dep.Verdict, dep.Overall)
	}
}

func TestGradeBoundaries(t *testing.T) {
	cases := map[int]report.Grade{90: report.GradeA, 72: report.GradeB, 60: report.GradeC, 41: report.GradeD, 10: report.GradeF}
	for sc, want := range cases {
		if got := report.GradeFor(sc); got != want {
			t.Errorf("GradeFor(%d) = %s, want %s", sc, got, want)
		}
	}
}

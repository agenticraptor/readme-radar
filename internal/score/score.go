// Package score combines the per-axis findings (maintenance, permissions,
// telemetry, footprint) into a single 0–100 number, a letter grade, and a
// one-line verdict. The weighting deliberately favors the safety axes:
// a leanly-built but abandoned or install-script-laden package should not pass.
package score

import (
	"fmt"

	"github.com/agenticraptor/readme-radar/internal/humanize"
	"github.com/agenticraptor/readme-radar/internal/report"
)

// axis weights (sum to 1.0).
const (
	wHealth      = 0.35
	wPermissions = 0.30
	wTelemetry   = 0.20
	wFootprint   = 0.15
)

// Footprint scores install size and direct-dependency count, returning the
// 0–100 score and the factors to attach to the SizeResult.
func Footprint(s report.SizeResult) (int, []report.Factor) {
	if !s.Known {
		return 70, []report.Factor{{
			Name: "Install size", Level: report.LevelInfo, Status: report.LevelInfo.String(),
			Detail: "not published to a registry — size estimated from the repo",
		}}
	}
	var base int
	var sizeLvl report.Level
	switch b := s.InstallBytes; {
	case b <= 0:
		base, sizeLvl = 70, report.LevelInfo
	case b < 100<<10:
		base, sizeLvl = 100, report.LevelGood
	case b < 500<<10:
		base, sizeLvl = 90, report.LevelGood
	case b < 2<<20:
		base, sizeLvl = 78, report.LevelGood
	case b < 10<<20:
		base, sizeLvl = 60, report.LevelWarn
	case b < 40<<20:
		base, sizeLvl = 42, report.LevelWarn
	default:
		base, sizeLvl = 25, report.LevelBad
	}

	switch {
	case s.DirectDeps == 0:
	case s.DirectDeps <= 5:
	case s.DirectDeps <= 15:
		base -= 5
	case s.DirectDeps <= 30:
		base -= 12
	default:
		base -= 20
	}

	factors := []report.Factor{{
		Name: "Install size", Level: sizeLvl, Status: sizeLvl.String(),
		Detail: sizeDetail(s),
	}}
	depLvl := report.LevelGood
	if s.DirectDeps > 15 {
		depLvl = report.LevelWarn
	}
	if s.DirectDeps > 0 {
		factors = append(factors, report.Factor{
			Name: "Direct dependencies", Level: depLvl, Status: depLvl.String(),
			Detail: fmt.Sprintf("%d direct dependenc%s", s.DirectDeps, plural(s.DirectDeps)),
		})
	}
	return clamp(base), factors
}

func sizeDetail(s report.SizeResult) string {
	src := s.Source
	if src == "" {
		src = "registry"
	}
	if s.InstallBytes <= 0 {
		return "size not reported by " + src
	}
	return fmt.Sprintf("%s unpacked (%s)", humanize.Bytes(s.InstallBytes), src)
}

// Combine produces the overall verdict. SizeResult.Score must already be set
// (via Footprint). Certain critical findings gate (cap) the overall score so a
// pristine, popular repo can't "buy back" a malicious install script with its
// maintenance halo — surfacing exactly that is the point of the tool.
func Combine(h report.HealthResult, t report.TelemetryResult, p report.PermissionsResult, s report.SizeResult) report.ScoreResult {
	overall := int(float64(h.Score)*wHealth +
		float64(p.Score)*wPermissions +
		float64(t.Score)*wTelemetry +
		float64(s.Score)*wFootprint + 0.5)
	overall = clamp(overall)

	// Gates: red-flag findings cap the headline verdict.
	capAt := 100
	if h.Deprecated != "" {
		capAt = mini(capAt, 50) // the author says stop using it → at most "caution"
	}
	if hasBad(p.Capabilities) {
		capAt = mini(capAt, 64) // runs code at install time → at most "caution"
	}
	if p.Score < 45 {
		capAt = mini(capAt, 44) // install script fetches/executes remote code → "avoid"
	}
	if t.Level == report.LevelBad {
		capAt = mini(capAt, 69) // embedded product analytics → at most "caution"
	}
	if overall > capAt {
		overall = capAt
	}

	axes := []report.Axis{
		axis("Maintenance", h.Score),
		axis("Permissions", p.Score),
		axis("Telemetry", t.Score),
		axis("Footprint", s.Score),
	}

	return report.ScoreResult{
		Overall:  overall,
		Grade:    report.GradeFor(overall),
		Verdict:  report.VerdictFor(overall),
		Headline: headline(overall, h, p, t),
		Axes:     axes,
	}
}

func axis(name string, sc int) report.Axis {
	return report.Axis{Name: name, Score: sc, Grade: report.GradeFor(sc), Detail: ""}
}

// headline picks a short, specific summary line based on the weakest axis.
func headline(overall int, h report.HealthResult, p report.PermissionsResult, t report.TelemetryResult) string {
	switch report.VerdictFor(overall) {
	case report.VerdictTrust:
		return "Healthy, lean, and quiet — safe to adopt with the usual review."
	case report.VerdictCaution:
		switch {
		case p.Score < 60:
			return "Works, but it runs code at install time — read those scripts first."
		case h.Score < 55:
			return "Useful, but maintenance looks thin — check how recently it shipped."
		case t.Score < 70:
			return "Fine, but it phones home — confirm the telemetry is acceptable to you."
		default:
			return "Mostly fine, with a couple of things worth a closer look."
		}
	default:
		switch {
		case p.Score < 45:
			return "High-risk install behavior — vet it carefully before adding it."
		case h.Score < 40:
			return "Looks abandoned or archived — depending on it is risky."
		default:
			return "Several red flags — look closely before you trust this one."
		}
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

func mini(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func plural(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}

func clamp(n int) int {
	if n < 0 {
		return 0
	}
	if n > 100 {
		return 100
	}
	return n
}

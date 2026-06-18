// Package health turns raw GitHub repository facts into a 0–100 maintenance
// score plus a short list of human-readable factors. The scoring is fully
// deterministic so the same repo always yields the same number.
package health

import (
	"fmt"
	"time"

	"github.com/agenticraptor/readme-radar/internal/report"
)

// component weights (sum to 100).
const (
	wRecency      = 32
	wContributors = 20
	wReleases     = 12
	wLicense      = 10
	wMaturity     = 10
	wPopularity   = 8
	wIssues       = 8
)

// Assess scores how actively and safely a repository is maintained. now is
// injected so results are reproducible in tests.
func Assess(info report.RepoInfo, now time.Time) report.HealthResult {
	var factors []report.Factor

	recency, recencyDetail := recencyScore(info.PushedAt, now)
	factors = append(factors, factor("Last commit", recencyLevel(info.PushedAt, now), recencyDetail))

	contrib := contributorScore(info.Contributors)
	if info.Contributors > 0 {
		factors = append(factors, factor("Contributors", contribLevel(info.Contributors),
			fmt.Sprintf("%d contributor%s", info.Contributors, plural(info.Contributors))))
	}

	releases := releaseScore(info)
	if info.Releases > 0 {
		factors = append(factors, factor("Releases", report.LevelGood,
			fmt.Sprintf("%d release%s, latest %s", info.Releases, plural(info.Releases), tagOrAge(info, now))))
	} else {
		factors = append(factors, factor("Releases", report.LevelInfo, "no tagged releases"))
	}

	license := 0.0
	if info.License != "" && info.License != "none" {
		license = 1
		factors = append(factors, factor("License", report.LevelGood, info.License))
	} else {
		factors = append(factors, factor("License", report.LevelWarn, "no license detected — reuse may be restricted"))
	}

	maturity := maturityScore(info.CreatedAt, info.PushedAt, now)
	popularity := popularityScore(info.Stars)
	issues, issuesDetail := issueScore(info.OpenIssues, info.Stars)
	if issuesDetail != "" {
		factors = append(factors, factor("Open issues", issuesLevel(info.OpenIssues, info.Stars), issuesDetail))
	}

	raw := recency*wRecency +
		contrib*wContributors +
		releases*wReleases +
		license*wLicense +
		maturity*wMaturity +
		popularity*wPopularity +
		issues*wIssues
	score := int(raw + 0.5)

	// Hard overrides for archived / disabled repos.
	switch {
	case info.Disabled:
		score = minInt(score, 15)
		factors = append([]report.Factor{factor("Status", report.LevelBad, "repository is disabled")}, factors...)
	case info.Archived:
		score = minInt(score, 25)
		factors = append([]report.Factor{factor("Status", report.LevelBad, "repository is archived (read-only, unmaintained)")}, factors...)
	}
	if info.Fork {
		score = clamp(score - 5)
		factors = append(factors, factor("Fork", report.LevelInfo, "this is a fork, not the upstream project"))
	}

	score = clamp(score)
	return report.HealthResult{
		Score:      score,
		Factors:    factors,
		Maintained: !info.Archived && !info.Disabled && monthsSince(info.PushedAt, now) <= 6,
	}
}

func recencyScore(pushed, now time.Time) (float64, string) {
	if pushed.IsZero() {
		return 0.3, "unknown last-commit date"
	}
	d := now.Sub(pushed)
	days := int(d.Hours() / 24)
	switch {
	case days <= 30:
		return 1.0, fmt.Sprintf("active — last push %s ago", humanDays(days))
	case days <= 90:
		return 0.85, fmt.Sprintf("last push %s ago", humanDays(days))
	case days <= 180:
		return 0.65, fmt.Sprintf("last push %s ago", humanDays(days))
	case days <= 365:
		return 0.4, fmt.Sprintf("last push %s ago", humanDays(days))
	case days <= 730:
		return 0.2, fmt.Sprintf("stale — last push %s ago", humanDays(days))
	default:
		return 0.05, fmt.Sprintf("dormant — last push %s ago", humanDays(days))
	}
}

func recencyLevel(pushed, now time.Time) report.Level {
	switch m := monthsSince(pushed, now); {
	case pushed.IsZero():
		return report.LevelInfo
	case m <= 3:
		return report.LevelGood
	case m <= 12:
		return report.LevelWarn
	default:
		return report.LevelBad
	}
}

func contributorScore(n int) float64 {
	switch {
	case n <= 0:
		return 0.5 // unknown
	case n == 1:
		return 0.35
	case n <= 3:
		return 0.6
	case n <= 9:
		return 0.85
	default:
		return 1.0
	}
}

func contribLevel(n int) report.Level {
	switch {
	case n == 1:
		return report.LevelWarn // bus factor of one
	case n <= 3:
		return report.LevelInfo
	default:
		return report.LevelGood
	}
}

func releaseScore(info report.RepoInfo) float64 {
	if info.Releases <= 0 {
		return 0.4
	}
	if info.Releases >= 5 {
		return 1.0
	}
	return 0.6 + 0.08*float64(info.Releases)
}

func maturityScore(created, pushed, now time.Time) float64 {
	if created.IsZero() {
		return 0.5
	}
	ageDays := int(now.Sub(created).Hours() / 24)
	active := !pushed.IsZero() && monthsSince(pushed, now) <= 12
	switch {
	case ageDays < 60:
		return 0.4 // brand new, unproven
	case ageDays < 180:
		return 0.6
	case active:
		return 1.0 // mature and still active
	default:
		return 0.7
	}
}

func popularityScore(stars int) float64 {
	switch {
	case stars >= 5000:
		return 1.0
	case stars >= 1000:
		return 0.9
	case stars >= 200:
		return 0.75
	case stars >= 50:
		return 0.6
	case stars >= 10:
		return 0.45
	default:
		return 0.3
	}
}

func issueScore(open, stars int) (float64, string) {
	if open == 0 {
		return 0.7, ""
	}
	denom := stars + open
	if denom == 0 {
		return 0.6, fmt.Sprintf("%d open", open)
	}
	ratio := float64(open) / float64(denom)
	detail := fmt.Sprintf("%d open issues", open)
	switch {
	case ratio < 0.03:
		return 1.0, detail
	case ratio < 0.06:
		return 0.8, detail
	case ratio < 0.12:
		return 0.6, detail
	case ratio < 0.25:
		return 0.4, detail
	default:
		return 0.25, detail + " (high relative to stars)"
	}
}

func issuesLevel(open, stars int) report.Level {
	if open == 0 {
		return report.LevelInfo
	}
	denom := stars + open
	if denom > 0 && float64(open)/float64(denom) >= 0.25 {
		return report.LevelWarn
	}
	return report.LevelInfo
}

// --- small helpers ---

func factor(name string, lvl report.Level, detail string) report.Factor {
	return report.Factor{Name: name, Level: lvl, Status: lvl.String(), Detail: detail}
}

func monthsSince(t, now time.Time) int {
	if t.IsZero() {
		return 9999
	}
	return int(now.Sub(t).Hours() / 24 / 30)
}

func humanDays(days int) string {
	switch {
	case days <= 1:
		return "a day"
	case days < 30:
		return fmt.Sprintf("%d days", days)
	case days < 365:
		return fmt.Sprintf("%d months", days/30)
	default:
		y := days / 365
		return fmt.Sprintf("%d year%s", y, plural(y))
	}
}

func tagOrAge(info report.RepoInfo, now time.Time) string {
	if info.LatestRelease != "" {
		if !info.LatestTagAt.IsZero() {
			return fmt.Sprintf("%s (%s ago)", info.LatestRelease, humanDays(int(now.Sub(info.LatestTagAt).Hours()/24)))
		}
		return info.LatestRelease
	}
	return "—"
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
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

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

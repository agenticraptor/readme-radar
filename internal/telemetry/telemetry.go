// Package telemetry estimates how "phone-home-y" a package looks by matching
// its declared dependencies (and a light scan of its manifest text) against
// curated lists of analytics, error-reporting, and network libraries. It is a
// best-effort heuristic, not a network sniffer — a clean result is not a
// guarantee of zero telemetry.
package telemetry

import (
	"sort"
	"strings"

	"github.com/agenticraptor/readme-radar/internal/report"
)

// Input is the evidence telemetry analysis reasons over.
type Input struct {
	Deps  []string          // declared dependency names
	Files map[string][]byte // raw manifest/source snippets for a light text scan
}

// strong product-analytics / session-recording SDKs (most invasive).
var analyticsLibs = []string{
	"segment", "analytics-node", "@segment/", "@segment/analytics-node",
	"mixpanel", "mixpanel-browser", "amplitude", "@amplitude/",
	"posthog-node", "posthog-js", "posthog", "@posthog/",
	"rudder-sdk-node", "@rudderstack/", "heap-api", "heap",
	"fullstory", "@fullstory/", "logrocket", "hotjar",
	"@google-analytics/", "universal-analytics", "react-ga", "react-ga4", "gtag",
	"segment-analytics-python", "analytics-python",
}

// error/perf telemetry (less invasive, still phones home).
var errorLibs = []string{
	"@sentry/", "sentry", "sentry-sdk", "@bugsnag/", "bugsnag",
	"rollbar", "raygun4js", "appsignal", "newrelic", "new-relic",
	"elastic-apm-node", "elastic-apm", "dd-trace", "@datadog/", "ddtrace",
	"opentelemetry", "@opentelemetry/",
}

// generic HTTP clients — informational context, not inherently bad.
var networkLibs = []string{
	"axios", "node-fetch", "cross-fetch", "got", "request", "superagent",
	"undici", "phin", "needle", "ky",
	"requests", "httpx", "urllib3", "aiohttp", "httplib2", "treq",
}

// telemetry domains/strings to catch in manifest text (for non-package repos).
var telemetryStrings = []string{
	"segment.io", "mixpanel.com", "google-analytics.com", "googletagmanager.com",
	"sentry.io", "getsentry", "bugsnag.com", "amplitude.com", "posthog.com",
	"app.posthog", "rudderlabs", "fullstory.com", "logrocket.com",
}

// Analyze produces a telemetry verdict.
func Analyze(in Input) report.TelemetryResult {
	analytics := matchAll(in.Deps, analyticsLibs)
	errs := matchAll(in.Deps, errorLibs)
	nets := matchAll(in.Deps, networkLibs)

	// Light manifest text scan for explicit telemetry endpoints.
	textHits := scanText(in.Files, telemetryStrings)

	score := 100
	var factors []report.Factor

	if len(analytics) > 0 {
		score -= 30 * len(analytics)
		factors = append(factors, fac("Product analytics", report.LevelBad,
			"bundles "+joinShort(analytics)))
	}
	if len(errs) > 0 {
		score -= 12 * len(errs)
		factors = append(factors, fac("Error/perf telemetry", report.LevelWarn,
			"includes "+joinShort(errs)))
	}
	if len(textHits) > 0 {
		score -= 12
		factors = append(factors, fac("Telemetry endpoints", report.LevelWarn,
			"manifest references "+joinShort(textHits)))
	}
	if len(nets) > 0 {
		factors = append(factors, fac("Network client", report.LevelInfo,
			"uses "+joinShort(nets)+" (normal for many libraries)"))
	}
	if len(analytics) == 0 && len(errs) == 0 && len(textHits) == 0 {
		factors = append(factors, fac("Telemetry", report.LevelGood,
			"no known analytics or telemetry SDKs in its dependencies"))
	}

	score = clamp(score)
	lvl := levelFor(score, len(analytics), len(errs)+len(textHits))
	return report.TelemetryResult{
		Score:         score,
		Level:         lvl,
		Status:        lvl.String(),
		AnalyticsLibs: dedupeSort(append(analytics, textHits...)),
		NetworkLibs:   dedupeSort(append(nets, errs...)),
		Factors:       factors,
	}
}

// levelFor maps the score and signal counts onto a traffic light. Any embedded
// product-analytics SDK is treated as a red flag; milder telemetry (error
// reporting, endpoint strings) caps the result at "warn".
func levelFor(score, analytics, otherSignals int) report.Level {
	switch {
	case analytics >= 1 || score < 55:
		return report.LevelBad
	case otherSignals > 0 || score < 85:
		return report.LevelWarn
	default:
		return report.LevelGood
	}
}

// matchAll returns the patterns that any dependency matches. A pattern ending
// in "/" matches by prefix (npm scopes); otherwise it matches the whole name.
func matchAll(deps, patterns []string) []string {
	var hits []string
	seen := map[string]bool{}
	for _, d := range deps {
		name := strings.ToLower(strings.TrimSpace(d))
		for _, p := range patterns {
			if matches(name, p) && !seen[p] {
				seen[p] = true
				hits = append(hits, strings.TrimSuffix(p, "/"))
			}
		}
	}
	return hits
}

func matches(name, pattern string) bool {
	if strings.HasSuffix(pattern, "/") {
		return strings.HasPrefix(name, pattern)
	}
	return name == pattern
}

func scanText(files map[string][]byte, needles []string) []string {
	var hits []string
	seen := map[string]bool{}
	for _, b := range files {
		low := strings.ToLower(string(b))
		for _, n := range needles {
			if strings.Contains(low, n) && !seen[n] {
				seen[n] = true
				hits = append(hits, n)
			}
		}
	}
	return hits
}

func joinShort(items []string) string {
	if len(items) <= 3 {
		return strings.Join(items, ", ")
	}
	return strings.Join(items[:3], ", ") + " and more"
}

func dedupeSort(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

func fac(name string, lvl report.Level, detail string) report.Factor {
	return report.Factor{Name: name, Level: lvl, Status: lvl.String(), Detail: detail}
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

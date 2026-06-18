// Package narrate turns a computed Report into a short, plain-English verdict.
// When a model is available it writes the blurb; otherwise a deterministic
// offline summary is generated from the same facts. The model only ever
// *describes* facts readme-radar already computed — it never scores or decides.
package narrate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/agenticraptor/readme-radar/internal/humanize"
	"github.com/agenticraptor/readme-radar/internal/llm"
	"github.com/agenticraptor/readme-radar/internal/report"
	"github.com/agenticraptor/readme-radar/internal/textutil"
)

const systemPrompt = `You write the one-paragraph "should I trust this?" blurb for readme-radar,
a tool that vets a software package before someone installs it.

Rules:
- Use ONLY the facts provided. Never invent versions, numbers, or behavior.
- The verdict and grade are already decided; reflect them, don't overrule them.
- Be concrete and calm. No hype, no fear-mongering, no marketing.
- 2-3 sentences for "summary": what the project is, then whether to trust it and why.
- Up to 3 short "bullets": the specific things that drove the verdict.
Respond with ONLY a JSON object: {"summary": string, "bullets": [string, ...]}.`

// Narrate produces the verdict narrative. If client is nil or the model errors,
// it returns a deterministic offline narrative — scanning never fails on this.
func Narrate(ctx context.Context, rep report.Report, client llm.Client) report.Narrative {
	if client == nil {
		return offline(rep)
	}
	out, err := client.Complete(ctx, llm.Request{
		System:      systemPrompt,
		Prompt:      facts(rep),
		MaxTokens:   600,
		Temperature: 0.2,
	})
	if err != nil {
		n := offline(rep)
		n.Source = n.Source + " (model unavailable: " + short(err.Error()) + ")"
		return n
	}
	n, ok := parse(out)
	if !ok || strings.TrimSpace(n.Summary) == "" {
		return offline(rep)
	}
	n.Source = client.Name()
	return n
}

// facts renders the report into a compact, model-friendly brief.
func facts(rep report.Report) string {
	var b strings.Builder
	t := rep.Target
	fmt.Fprintf(&b, "Target: %s (%s)\n", t.Slug(), t.Ecosystem)
	if rep.Repo.Description != "" {
		fmt.Fprintf(&b, "Description: %s\n", rep.Repo.Description)
	}
	fmt.Fprintf(&b, "Verdict: %s (overall %d/100, grade %s)\n", rep.Score.Verdict, rep.Score.Overall, rep.Score.Grade)
	for _, a := range rep.Score.Axes {
		fmt.Fprintf(&b, "- %s: %d/100 (%s)\n", a.Name, a.Score, a.Grade)
	}
	if rep.Repo.FullName != "" {
		fmt.Fprintf(&b, "Repo: %s stars, %d contributors, %d releases, license %s, primary language %s\n",
			humanize.Count(rep.Repo.Stars), rep.Repo.Contributors, rep.Repo.Releases, dash(rep.Repo.License), dash(rep.Repo.PrimaryLang))
	}
	if rep.Size.Known {
		fmt.Fprintf(&b, "Install size: %s unpacked, %d direct dependencies (%s)\n",
			humanize.Bytes(rep.Size.InstallBytes), rep.Size.DirectDeps, rep.Size.Source)
	}
	writeFactors(&b, "Maintenance notes", rep.Health.Factors)
	writeFactors(&b, "Telemetry notes", rep.Telemetry.Factors)
	writeFactors(&b, "Permission notes", rep.Permissions.Capabilities)
	if len(rep.Permissions.InstallScripts) > 0 {
		fmt.Fprintf(&b, "Install scripts: %s\n", strings.Join(rep.Permissions.InstallScripts, " | "))
	}
	return b.String()
}

func writeFactors(b *strings.Builder, title string, fs []report.Factor) {
	if len(fs) == 0 {
		return
	}
	fmt.Fprintf(b, "%s:\n", title)
	for _, f := range fs {
		fmt.Fprintf(b, "  [%s] %s: %s\n", f.Level.String(), f.Name, f.Detail)
	}
}

// parse extracts the {"summary","bullets"} object, tolerating code fences and
// surrounding prose.
func parse(s string) (report.Narrative, bool) {
	s = strings.TrimSpace(s)
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start < 0 || end <= start {
		return report.Narrative{}, false
	}
	var raw struct {
		Summary string   `json:"summary"`
		Bullets []string `json:"bullets"`
	}
	if err := json.Unmarshal([]byte(s[start:end+1]), &raw); err != nil {
		return report.Narrative{}, false
	}
	return report.Narrative{Summary: textutil.Safe(raw.Summary), Bullets: clean(raw.Bullets)}, true
}

// offline builds a deterministic narrative from the facts alone.
func offline(rep report.Report) report.Narrative {
	var sum strings.Builder
	name := rep.Target.Slug()
	if d := rep.Repo.Description; d != "" {
		fmt.Fprintf(&sum, "%s — %s. ", name, strings.TrimRight(d, "."))
	} else {
		fmt.Fprintf(&sum, "%s. ", name)
	}
	sum.WriteString(rep.Score.Headline)

	var bullets []string
	if rep.Repo.FullName != "" && len(rep.Score.Axes) > 0 {
		bullets = append(bullets, fmt.Sprintf("Maintenance %s: %s",
			rep.Score.Axes[0].Grade, firstDetail(rep.Health.Factors, "active development")))
	}
	if rep.Size.Known {
		bullets = append(bullets, fmt.Sprintf("Footprint: %s unpacked, %d direct deps",
			humanize.Bytes(rep.Size.InstallBytes), rep.Size.DirectDeps))
	}
	if len(rep.Permissions.InstallScripts) > 0 {
		bullets = append(bullets, "Runs install scripts: "+rep.Permissions.InstallScripts[0])
	} else if rep.Target.Ecosystem == report.EcosystemNPM {
		bullets = append(bullets, "No install scripts")
	}
	if len(rep.Telemetry.AnalyticsLibs) > 0 {
		bullets = append(bullets, "Telemetry: "+strings.Join(rep.Telemetry.AnalyticsLibs, ", "))
	}
	return report.Narrative{Summary: strings.TrimSpace(sum.String()), Bullets: bullets, Source: "offline"}
}

func firstDetail(fs []report.Factor, fallback string) string {
	if len(fs) > 0 {
		return fs[0].Detail
	}
	return fallback
}

func clean(in []string) []string {
	var out []string
	for _, s := range in {
		if s = textutil.Safe(s); s != "" {
			out = append(out, s)
		}
		if len(out) >= 3 {
			break
		}
	}
	return out
}

func dash(s string) string {
	if s == "" || s == "none" {
		return "—"
	}
	return s
}

func short(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 60 {
		return s[:60] + "…"
	}
	return s
}

package render

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/agenticraptor/readme-radar/internal/humanize"
	"github.com/agenticraptor/readme-radar/internal/report"
)

// card width (inner content, excluding border + padding).
const innerWidth = 60

// grade → accent color.
func gradeColor(g report.Grade) lipgloss.Color {
	switch g {
	case report.GradeA:
		return "#22c55e"
	case report.GradeB:
		return "#84cc16"
	case report.GradeC:
		return "#eab308"
	case report.GradeD:
		return "#f97316"
	default:
		return "#ef4444"
	}
}

// card renders the verdict card. When colorize is false it emits the same
// layout with no ANSI color (used for --no-color and the plain format).
func card(rep report.Report, colorize bool) string {
	accent := gradeColor(rep.Score.Grade)
	col := func(s string, c lipgloss.Color) string {
		if !colorize {
			return s
		}
		return lipgloss.NewStyle().Foreground(c).Render(s)
	}
	bold := func(s string) string {
		if !colorize {
			return s
		}
		return lipgloss.NewStyle().Bold(true).Render(s)
	}
	dim := func(s string) string {
		if !colorize {
			return s
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#94a3b8")).Render(s)
	}

	// Header: title left, target right.
	title := "📡 " + bold("readme-radar")
	loc := dim(locator(rep.Target))
	header := spread(title, loc, innerWidth)

	// Verdict line: big grade chip + verdict words + score.
	var chip string
	if colorize {
		chip = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0b0f17")).
			Background(accent).Padding(0, 1).Render(string(rep.Score.Grade))
	} else {
		chip = "[ " + string(rep.Score.Grade) + " ]"
	}
	verdict := bold(col(strings.ToUpper(string(rep.Score.Verdict)), accent))
	scoreStr := dim(fmt.Sprintf("%d / 100", rep.Score.Overall))
	left := chip + "  " + verdict
	verdictLine := spread(left, scoreStr, innerWidth)

	// Axis bars.
	var axes []string
	for _, a := range rep.Score.Axes {
		axes = append(axes, axisRow(a, colorize, col))
	}

	// Facts strip.
	facts := dim(factStrip(rep))

	// Narrative.
	body := []string{header, "", verdictLine, "", strings.Join(axes, "\n")}
	if facts != "" {
		body = append(body, "", facts)
	}
	if rep.Narrative.Summary != "" {
		wrapped := lipgloss.NewStyle().Width(innerWidth).Render(rep.Narrative.Summary)
		body = append(body, "", wrapped)
	}
	for _, bl := range rep.Narrative.Bullets {
		body = append(body, dim("• ")+lipgloss.NewStyle().Width(innerWidth-2).Render(bl))
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 2).
		Width(innerWidth + 4)
	if colorize {
		box = box.BorderForeground(accent)
	}
	out := box.Render(strings.Join(body, "\n"))

	foot := dim(fmt.Sprintf("  scanned %s · weighs maintenance · permissions · telemetry · footprint",
		scannedAgo(rep.GeneratedAt)))
	if rep.Narrative.Source != "" && rep.Narrative.Source != "offline" {
		foot = dim(fmt.Sprintf("  scanned %s · narrated by %s", scannedAgo(rep.GeneratedAt), rep.Narrative.Source))
	}

	var notes strings.Builder
	for _, w := range rep.Warnings {
		warn := "  ! " + w
		if colorize {
			warn = lipgloss.NewStyle().Foreground(lipgloss.Color("#eab308")).Render("  ! ") +
				dim(lipgloss.NewStyle().Width(innerWidth).Render(w))
		}
		notes.WriteString("\n" + warn)
	}
	return out + "\n" + foot + notes.String() + "\n"
}

func axisRow(a report.Axis, colorize bool, col func(string, lipgloss.Color) string) string {
	name := pad(a.Name, 12)
	grade := col(string(a.Grade), gradeColor(a.Grade))
	bar := barFor(a.Score, colorize, gradeColor(a.Grade))
	return fmt.Sprintf("%s %s  %s  %s", name, grade, bar, pad(fmt.Sprintf("%d", a.Score), 3))
}

// barFor renders a 10-cell meter.
func barFor(score int, colorize bool, c lipgloss.Color) string {
	filled := (score + 5) / 10
	if filled > 10 {
		filled = 10
	}
	if filled < 0 {
		filled = 0
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", 10-filled)
	if !colorize {
		return bar
	}
	return lipgloss.NewStyle().Foreground(c).Render(bar)
}

func factStrip(rep report.Report) string {
	var parts []string
	if rep.Repo.FullName != "" {
		parts = append(parts, "★ "+humanize.Count(rep.Repo.Stars))
	}
	if rep.Size.Known {
		parts = append(parts, "⬇ "+humanize.Bytes(rep.Size.InstallBytes))
	}
	if rep.Size.DirectDeps > 0 {
		parts = append(parts, fmt.Sprintf("⬚ %d deps", rep.Size.DirectDeps))
	}
	if l := rep.Repo.License; l != "" && l != "none" {
		parts = append(parts, "⚖ "+l)
	}
	if a := ageOf(rep); a != "" {
		parts = append(parts, "⟳ "+a)
	}
	return strings.Join(parts, "   ")
}

func locator(t report.Target) string {
	switch t.Ecosystem {
	case report.EcosystemNPM:
		return "npm:" + t.Package
	case report.EcosystemPyPI:
		return "pypi:" + t.Package
	default:
		return t.Slug()
	}
}

func ageOf(rep report.Report) string {
	if rep.Repo.PushedAt.IsZero() {
		return ""
	}
	return humanAgo(rep.Repo.PushedAt, rep.GeneratedAt)
}

func humanAgo(t, now time.Time) string {
	d := now.Sub(t)
	days := int(d.Hours() / 24)
	switch {
	case days <= 0:
		return "today"
	case days == 1:
		return "yesterday"
	case days < 30:
		return fmt.Sprintf("%dd ago", days)
	case days < 365:
		return fmt.Sprintf("%dmo ago", days/30)
	default:
		return fmt.Sprintf("%dy ago", days/365)
	}
}

func scannedAgo(t time.Time) string {
	if t.IsZero() {
		return "just now"
	}
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	return humanAgo(t, time.Now())
}

// spread places left and right text at the edges of a width-w line.
func spread(left, right string, w int) string {
	gap := w - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

func pad(s string, w int) string {
	if lipgloss.Width(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-lipgloss.Width(s))
}

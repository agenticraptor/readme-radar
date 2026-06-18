// Package card renders a Report as a shareable "verdict card" — the most
// screenshot-able artifact readme-radar produces. The SVG renderer is pure
// stdlib; PNG (png.go) rasterizes the same layout with golang.org/x/image.
package card

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/agenticraptor/readme-radar/internal/humanize"
	"github.com/agenticraptor/readme-radar/internal/report"
)

// Canvas dimensions (logical pixels). PNG renders at a scale multiple of these.
const (
	width  = 860
	height = 470
)

// palette (dark theme).
var (
	bgColor     = "#0b0f17"
	cardColor   = "#111827"
	strokeColor = "#1f2937"
	textColor   = "#e5e7eb"
	dimColor    = "#94a3b8"
	trackColor  = "#1f2937"
)

// model is the layout-ready view of a Report, shared by SVG and PNG.
type model struct {
	target  string
	grade   string
	accent  string
	verdict string
	score   int
	axes    []axisVM
	facts   string
	summary []string // up to 2 wrapped lines
	footer  string
}

type axisVM struct {
	name  string
	score int
	grade string
	color string
}

func gradeHex(g report.Grade) string {
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

func build(rep report.Report) model {
	m := model{
		target:  sanitize(locator(rep.Target)),
		grade:   string(rep.Score.Grade),
		accent:  gradeHex(rep.Score.Grade),
		verdict: strings.ToUpper(string(rep.Score.Verdict)),
		score:   rep.Score.Overall,
		facts:   sanitize(facts(rep)),
		summary: wrap(sanitize(summaryText(rep)), 84, 2),
		footer:  "github.com/agenticraptor/readme-radar",
	}
	for _, a := range rep.Score.Axes {
		m.axes = append(m.axes, axisVM{name: a.Name, score: a.Score, grade: string(a.Grade), color: gradeHex(a.Grade)})
	}
	return m
}

// SVG renders the verdict card as a standalone SVG document.
func SVG(rep report.Report) string {
	m := build(rep)
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" font-family="-apple-system,Segoe UI,Roboto,Helvetica,Arial,sans-serif">`, width, height, width, height)

	// Background + card.
	fmt.Fprintf(&b, `<rect width="%d" height="%d" rx="20" fill="%s"/>`, width, height, bgColor)
	fmt.Fprintf(&b, `<rect x="14" y="14" width="%d" height="%d" rx="16" fill="%s" stroke="%s"/>`, width-28, height-28, cardColor, strokeColor)
	// Accent rail.
	fmt.Fprintf(&b, `<rect x="14" y="14" width="8" height="%d" rx="4" fill="%s"/>`, height-28, m.accent)

	// Header.
	radarIcon(&b, 50, 54, m.accent)
	fmt.Fprintf(&b, `<text x="78" y="62" fill="%s" font-size="28" font-weight="700">readme-radar</text>`, textColor)
	fmt.Fprintf(&b, `<text x="%d" y="62" fill="%s" font-size="18" text-anchor="end">%s</text>`, width-40, dimColor, esc(trunc(m.target, 52)))
	fmt.Fprintf(&b, `<line x1="40" y1="86" x2="%d" y2="86" stroke="%s"/>`, width-40, strokeColor)

	// Grade badge.
	fmt.Fprintf(&b, `<rect x="44" y="112" width="100" height="100" rx="18" fill="%s"/>`, m.accent)
	fmt.Fprintf(&b, `<text x="94" y="182" fill="#0b0f17" font-size="68" font-weight="800" text-anchor="middle">%s</text>`, m.grade)

	// Verdict + score + facts.
	fmt.Fprintf(&b, `<text x="166" y="146" fill="%s" font-size="30" font-weight="700">%s</text>`, m.accent, esc(m.verdict))
	fmt.Fprintf(&b, `<text x="166" y="178" fill="%s" font-size="20">%d / 100</text>`, dimColor, m.score)
	fmt.Fprintf(&b, `<text x="166" y="206" fill="%s" font-size="16">%s</text>`, dimColor, esc(m.facts))

	// Axis bars.
	y := 256
	for _, a := range m.axes {
		fmt.Fprintf(&b, `<text x="44" y="%d" fill="%s" font-size="16">%s</text>`, y+4, textColor, esc(a.name))
		fmt.Fprintf(&b, `<rect x="180" y="%d" width="500" height="12" rx="6" fill="%s"/>`, y-6, trackColor)
		fw := 500 * clamp(a.score) / 100
		if fw > 0 {
			fmt.Fprintf(&b, `<rect x="180" y="%d" width="%d" height="12" rx="6" fill="%s"/>`, y-6, fw, a.color)
		}
		fmt.Fprintf(&b, `<text x="700" y="%d" fill="%s" font-size="15" font-weight="700">%s</text>`, y+4, a.color, a.grade)
		fmt.Fprintf(&b, `<text x="724" y="%d" fill="%s" font-size="15">%d</text>`, y+4, dimColor, a.score)
		y += 34
	}

	// Narrative.
	ny := y + 14
	for _, line := range m.summary {
		fmt.Fprintf(&b, `<text x="44" y="%d" fill="%s" font-size="15">%s</text>`, ny, textColor, esc(line))
		ny += 22
	}

	// Footer.
	fmt.Fprintf(&b, `<text x="44" y="%d" fill="%s" font-size="13">%s</text>`, height-26, dimColor, esc(m.footer))
	fmt.Fprintf(&b, `<text x="%d" y="%d" fill="%s" font-size="13" text-anchor="end">grade %s · %s</text>`,
		width-40, height-26, dimColor, m.grade, esc(strings.ToLower(m.verdict)))

	b.WriteString(`</svg>`)
	return b.String()
}

// radarIcon draws a small concentric-ring "radar" glyph.
func radarIcon(b *strings.Builder, cx, cy int, accent string) {
	for i, r := range []int{16, 11, 6} {
		op := 0.4 + 0.3*float64(i)
		fmt.Fprintf(b, `<circle cx="%d" cy="%d" r="%d" fill="none" stroke="%s" stroke-opacity="%.2f" stroke-width="2"/>`, cx, cy, r, accent, op)
	}
	fmt.Fprintf(b, `<circle cx="%d" cy="%d" r="3" fill="%s"/>`, cx, cy, accent)
	fmt.Fprintf(b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="2"/>`, cx, cy, cx+14, cy-14, accent)
}

// --- shared text helpers ---

func locator(t report.Target) string {
	switch t.Ecosystem {
	case report.EcosystemNPM:
		return "npm: " + t.Package
	case report.EcosystemPyPI:
		return "pypi: " + t.Package
	default:
		return t.Slug()
	}
}

func facts(rep report.Report) string {
	var parts []string
	if rep.Repo.FullName != "" {
		parts = append(parts, humanize.Count(rep.Repo.Stars)+" stars")
	}
	if rep.Size.Known {
		parts = append(parts, humanize.Bytes(rep.Size.InstallBytes))
	}
	if rep.Size.DirectDeps > 0 {
		parts = append(parts, fmt.Sprintf("%d deps", rep.Size.DirectDeps))
	}
	if l := rep.Repo.License; l != "" && l != "none" {
		parts = append(parts, l)
	}
	return strings.Join(parts, "  ·  ")
}

func summaryText(rep report.Report) string {
	if s := strings.TrimSpace(rep.Narrative.Summary); s != "" {
		return s
	}
	return rep.Score.Headline
}

// wrap splits text into at most maxLines lines of at most width runes,
// breaking on spaces. The final line is ellipsized if text remains.
func wrap(s string, width, maxLines int) []string {
	words := strings.Fields(s)
	var lines []string
	var cur string
	for _, w := range words {
		if cur == "" {
			cur = w
		} else if len(cur)+1+len(w) <= width {
			cur += " " + w
		} else {
			lines = append(lines, cur)
			cur = w
			if len(lines) == maxLines {
				break
			}
		}
	}
	if len(lines) < maxLines && cur != "" {
		lines = append(lines, cur)
	}
	if len(lines) == maxLines {
		// signal truncation if we dropped words
		joined := strings.Join(lines, " ")
		if len(strings.Fields(joined)) < len(words) {
			rl := []rune(lines[maxLines-1])
			if len(rl) > width-1 {
				rl = rl[:width-1]
			}
			lines[maxLines-1] = strings.TrimRight(string(rl), " ") + "…"
		}
	}
	return lines
}

// sanitize removes emoji and pictographs (which the embedded card font can't
// render) while keeping ordinary punctuation like "—", "·", and "…". It then
// collapses any whitespace the removal left behind.
func sanitize(s string) string {
	var b strings.Builder
	for _, r := range s {
		if isPictograph(r) {
			continue
		}
		b.WriteRune(r)
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func isPictograph(r rune) bool {
	switch {
	case r >= 0x1F000 && r <= 0x1FAFF: // emoji & pictographs
		return true
	case r >= 0x2600 && r <= 0x27BF: // misc symbols & dingbats
		return true
	case r >= 0x2300 && r <= 0x23FF: // misc technical
		return true
	case r >= 0x1F1E6 && r <= 0x1F1FF: // regional indicators (flags)
		return true
	case r == 0x200D || (r >= 0xFE00 && r <= 0xFE0F) || r == 0x20E3: // ZWJ, variation selectors, keycap
		return true
	default:
		return false
	}
}

func trunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func esc(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
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

func hexRGBA(hex string) color.RGBA {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return color.RGBA{0xe5, 0xe7, 0xeb, 0xff}
	}
	var r, g, b uint8
	if _, err := fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b); err != nil {
		return color.RGBA{0xe5, 0xe7, 0xeb, 0xff}
	}
	return color.RGBA{r, g, b, 0xff}
}

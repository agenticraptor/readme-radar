// Package textutil sanitizes untrusted text. Repository descriptions, package
// summaries, and model output are attacker-controllable: a malicious repo could
// embed ANSI escape sequences to spoof or corrupt your terminal, or control
// characters that make a generated SVG invalid. Safe strips those before the
// text reaches any renderer.
package textutil

import (
	"regexp"
	"strings"
)

// ansiRe matches whole CSI sequences (e.g. "\x1b[31m") and OSC sequences
// (e.g. "\x1b]0;title\x07") so they're removed cleanly rather than leaving
// their parameter bytes behind as visible litter.
var ansiRe = regexp.MustCompile("\x1b\\[[0-9;?]*[ -/]*[@-~]|\x1b\\][^\x07\x1b]*(?:\x07|\x1b\\\\)")

// Safe removes ANSI escape sequences and control characters from untrusted
// text, converts any remaining whitespace runs to single spaces, and trims the
// result. Printable Unicode (including punctuation and non-Latin scripts) is
// preserved. This protects the terminal from spoofing/clobbering via a
// malicious repository description or package summary, and keeps generated SVG
// valid.
func Safe(s string) string {
	s = ansiRe.ReplaceAllString(s, "")
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\t' || r == '\n' || r == '\r':
			b.WriteByte(' ') // collapse later
		case r == 0x7f: // DEL
			continue
		case r < 0x20: // C0 controls, incl. ESC (0x1b) used by ANSI sequences
			continue
		case r >= 0x80 && r <= 0x9f: // C1 controls
			continue
		case r == 0x2028 || r == 0x2029: // line/paragraph separators
			b.WriteByte(' ')
		default:
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

package textutil

import "testing"

func TestSafeStripsControlAndANSI(t *testing.T) {
	cases := []struct{ in, want string }{
		{"hello", "hello"},
		{"a\x1b[31mred\x1b[0mb", "aredb"},         // ANSI SGR color sequences removed whole
		{"line1\nline2", "line1 line2"},           // newline -> space
		{"tab\there", "tab here"},                 // tab -> space
		{"bell\x07nul\x00del\x7f", "bellnuldel"},  // C0 + DEL removed
		{"  spaced   out  ", "spaced out"},        // whitespace collapsed + trimmed
		{"title\x1b]0;pwn\x07end", "titleend"},    // OSC window-title sequence removed whole
		{string(rune(0x9b)) + "31mCSI", "31mCSI"}, // C1 CSI byte (U+009B) removed
	}
	for _, c := range cases {
		if got := Safe(c.in); got != c.want {
			t.Errorf("Safe(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSafePreservesUnicode(t *testing.T) {
	in := "cafe — naive check 你好"
	if got := Safe(in); got != in {
		t.Errorf("Safe stripped legitimate Unicode: %q -> %q", in, got)
	}
}

func TestSafeNoEscapeSurvives(t *testing.T) {
	out := Safe("x\x1b\x1b\x1b[2J\x1b[1;1Hwiped")
	for _, r := range out {
		if r == 0x1b {
			t.Fatalf("ESC survived sanitization: %q", out)
		}
	}
}

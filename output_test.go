package assumpgo

import (
	"bytes"
	"strings"
	"testing"
)

func resultWith(assumptions ...Assumption) *Result {
	r := &Result{boolExpressionsCount: 3}
	r.assumptions = append(r.assumptions, assumptions...)
	return r
}

func TestPrettyOutputWithAssumptions(t *testing.T) {
	r := resultWith(
		Assumption{File: "a.go", Line: 2, Message: "if x != nil {"},
		Assumption{File: "longer/path.go", Line: 17, Message: "for running {"},
	)

	var buf bytes.Buffer
	if err := (PrettyOutput{}).Output(&buf, r); err != nil {
		t.Fatalf("Output: %v", err)
	}
	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	// The table lines (everything before the blank line) must all share the
	// same width, otherwise a column-width or border calculation has drifted.
	var tableLines []string
	for _, l := range lines {
		if l == "" {
			break
		}
		tableLines = append(tableLines, l)
	}
	if len(tableLines) != 6 {
		// 1 border + header + 1 separator + 2 rows + 1 border
		t.Fatalf("expected 6 table lines, got %d:\n%s", len(tableLines), out)
	}
	width := len(tableLines[0])
	for i, l := range tableLines {
		if len(l) != width {
			t.Errorf("table line %d width = %d, want %d:\n%q", i, len(l), width, l)
		}
	}

	if !strings.HasPrefix(tableLines[0], "-") || !strings.HasPrefix(tableLines[len(tableLines)-1], "-") {
		t.Error("table must be wrapped in dashed borders")
	}
	if !strings.HasPrefix(tableLines[2], "=") {
		t.Error("header must be followed by an = separator")
	}
	for _, want := range []string{"file", "line", "message", "a.go", "2", "if x != nil {", "longer/path.go", "17", "for running {"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}

	// There must be a blank line between the table and the summary.
	if !strings.Contains(out, "\n\n") {
		t.Error("expected a blank line before the summary")
	}
	if !strings.HasSuffix(out, "2 out of 3 boolean expressions are assumptions (67%)\n") {
		t.Errorf("unexpected summary line:\n%s", out)
	}
}

// TestPrettyOutputSingleAssumption guards the `AssumptionsCount() > 0` table
// boundary: a result with exactly one assumption must still render the table,
// not just collapse to the summary line.
func TestPrettyOutputSingleAssumption(t *testing.T) {
	r := &Result{boolExpressionsCount: 4}
	r.assumptions = append(r.assumptions, Assumption{File: "a.go", Line: 3, Message: "if x != nil {"})

	var buf bytes.Buffer
	if err := (PrettyOutput{}).Output(&buf, r); err != nil {
		t.Fatalf("Output: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "|") || !strings.Contains(out, "if x != nil {") {
		t.Errorf("expected a table for a single assumption:\n%s", out)
	}
	if !strings.HasSuffix(out, "1 out of 4 boolean expressions are assumptions (25%)\n") {
		t.Errorf("unexpected summary line:\n%s", out)
	}
}

func TestPrettyOutputWithoutAssumptions(t *testing.T) {
	r := &Result{boolExpressionsCount: 5}

	var buf bytes.Buffer
	if err := (PrettyOutput{}).Output(&buf, r); err != nil {
		t.Fatalf("Output: %v", err)
	}
	out := buf.String()

	if strings.Contains(out, "|") || strings.Contains(out, "-") {
		t.Errorf("expected no table when there are no assumptions:\n%s", out)
	}
	if out != "0 out of 5 boolean expressions are assumptions (0%)\n" {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestXMLOutput(t *testing.T) {
	r := resultWith(
		Assumption{File: "a.go", Line: 2, Message: "if x != nil {"},
		Assumption{File: "a.go", Line: 9, Message: "for running {"},
		Assumption{File: "b.go", Line: 4, Message: "!ready"},
	)

	var buf bytes.Buffer
	if err := (XMLOutput{}).Output(&buf, r); err != nil {
		t.Fatalf("Output: %v", err)
	}
	out := buf.String()

	for _, want := range []string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<checkstyle>`,
		`<file name="a.go">`,
		`<file name="b.go">`,
		`line="2"`,
		`line="9"`,
		`line="4"`,
		`severity="error"`,
		`message="if x != nil {"`,
		`message="!ready"`,
		`source="assumpgo"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("XML output missing %q:\n%s", want, out)
		}
	}

	// a.go groups two errors, and its <file> element must come before b.go's
	// (source order is preserved).
	if strings.Count(out, "<file ") != 2 {
		t.Errorf("expected 2 <file> elements, got %d:\n%s", strings.Count(out, "<file "), out)
	}
	if strings.Index(out, `name="a.go"`) > strings.Index(out, `name="b.go"`) {
		t.Error("expected a.go to be emitted before b.go")
	}
	if c := strings.Count(out, "<error "); c != 3 {
		t.Errorf("expected 3 <error> elements, got %d", c)
	}
}

func TestXMLOutputEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := (XMLOutput{}).Output(&buf, &Result{}); err != nil {
		t.Fatalf("Output: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "<checkstyle>") {
		t.Errorf("expected a checkstyle root element:\n%s", out)
	}
	if strings.Contains(out, "<file ") {
		t.Errorf("expected no <file> elements for an empty result:\n%s", out)
	}
}

func TestPrettyOutputMultibyteAlignment(t *testing.T) {
	r := resultWith(
		Assumption{File: "main.go", Line: 4, Message: "if dog != nil { // café ☕"},
		Assumption{File: "path/to/日本語.go", Line: 12, Message: "if cat != nil {"},
		Assumption{File: "short.go", Line: 99, Message: "if fish != nil { // pure ascii message longer"},
	)

	var buf bytes.Buffer
	if err := (PrettyOutput{}).Output(&buf, r); err != nil {
		t.Fatalf("Output: %v", err)
	}
	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	var tableLines []string
	for _, l := range lines {
		if l == "" {
			break
		}
		tableLines = append(tableLines, l)
	}
	if len(tableLines) != 7 {
		// 1 border + header + 1 separator + 3 rows + 1 border
		t.Fatalf("expected 7 table lines, got %d:\n%s", len(tableLines), out)
	}

	wantWidth := stringWidth(tableLines[0])
	for i, l := range tableLines {
		if got := stringWidth(l); got != wantWidth {
			t.Errorf("table line %d display width = %d, want %d:\n%q", i, got, wantWidth, l)
		}
	}

	// Verify column dividers align across all rows
	for i, l := range tableLines {
		if strings.HasPrefix(l, "-") || strings.HasPrefix(l, "=") {
			continue
		}
		// Count positions of | in visual columns
		col := 0
		var pipePositions []int
		for _, r := range l {
			if r == '|' {
				pipePositions = append(pipePositions, col)
			}
			col += runeWidth(r)
		}
		if len(pipePositions) != 4 {
			t.Errorf("line %d has %d pipes, want 4:\n%s", i, len(pipePositions), l)
		}
	}
}

func TestRuneWidth(t *testing.T) {
	tests := []struct {
		r    rune
		want int
	}{
		// Control and non-printable characters
		{0, 0},
		{31, 0},
		{0x7f, 0},
		{0x80, 0},
		{0x9f, 0},

		// Nonspacing and enclosing combining marks are zero columns
		{0x0301, 0}, // combining acute accent
		{0x0328, 0}, // combining ogonek (Mn)
		{0x20dd, 0}, // combining enclosing circle (Me)
		{0x05d0, 1}, // Hebrew letter: not a combining mark

		// Printable ASCII
		{32, 1},
		{'a', 1},
		{'~', 1},

		// Latin-1 Supplement (accented characters are 1 column)
		{0xa0, 1},
		{'é', 1},
		{'ü', 1},

		// Hangul Jamo boundaries
		{0x10ff, 1},
		{0x1100, 2},
		{0x11ff, 2},
		{0x1200, 1},

		// Misc Symbols and Dingbats boundaries (including ☕ = 0x2615)
		{0x25ff, 1},
		{0x2600, 2},
		{0x2615, 2},
		{0x27bf, 2},
		{0x27c0, 1},

		// CJK / Fullwidth boundaries
		{0x2e7f, 1},
		{0x2e80, 2},
		{'日', 2},
		{0xa4cf, 2},
		{0xa4d0, 1}, // Lisu is narrow
		{0xabff, 1},
		{0xac00, 2}, // Hangul syllables
		{0xd7af, 2},
		{0xd7b0, 1},
		{0xf8ff, 1}, // private use is narrow
		{0xf900, 2}, // CJK compatibility ideographs
		{0xfaff, 2},

		// Alphabetic Presentation Forms: Latin ligatures are one column
		{0xfb00, 1}, // ﬀ
		{'ﬁ', 1},
		{0xfb4f, 1},

		// Halfwidth and Fullwidth Forms: only the fullwidth halves are 2
		{0xff00, 1},
		{0xff01, 2}, // fullwidth exclamation mark
		{0xff60, 2},
		{0xff61, 1}, // halfwidth ideographic full stop
		{'ｶ', 1},    // halfwidth Katakana
		{0xffdc, 1},
		{0xffe0, 2}, // fullwidth cent sign
		{0xffee, 2},
		{0xffef, 1}, // unassigned, beyond the fullwidth signs
		{0xfff0, 1},

		// Emojis (SMP >= 0x1f000)
		{0x1efff, 1},
		{0x1f000, 2},
		{'🚀', 2},
	}

	for _, tt := range tests {
		if got := runeWidth(tt.r); got != tt.want {
			t.Errorf("runeWidth(%#x / %q) = %d, want %d", tt.r, tt.r, got, tt.want)
		}
	}
}

// TestPrettyOutputCombiningMarkAlignment guards the reported bug: a message
// containing a combining mark (here e + U+0301, two code points but one
// terminal column) must not shift its row's pipes out of alignment. Unlike
// TestPrettyOutputMultibyteAlignment, this test measures display width with an
// independent routine (not stringWidth, the code under test) so the check is
// not circular.
func TestPrettyOutputCombiningMarkAlignment(t *testing.T) {
	r := resultWith(
		Assumption{File: "combining.go", Line: 3, Message: "if value != nil { // é"},
		Assumption{File: "plain.go", Line: 9, Message: "if value != nil { // pure ascii message longer"},
	)

	var buf bytes.Buffer
	if err := (PrettyOutput{}).Output(&buf, r); err != nil {
		t.Fatalf("Output: %v", err)
	}
	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	var tableLines []string
	for _, l := range lines {
		if l == "" {
			break
		}
		tableLines = append(tableLines, l)
	}
	if len(tableLines) != 6 {
		// 1 border + header + 1 separator + 2 rows + 1 border
		t.Fatalf("expected 6 table lines, got %d:\n%s", len(tableLines), out)
	}

	// visualWidth counts terminal columns the way a terminal does for this
	// input: every rune is one column except the combining acute accent
	// (U+0301), which is zero columns.
	visualWidth := func(s string) int {
		w := 0
		for _, r := range s {
			if r != 0x0301 {
				w++
			}
		}
		return w
	}

	// The border is pure ASCII, so its byte length is its display width.
	wantWidth := len(tableLines[0])
	for i, l := range tableLines {
		if got := visualWidth(l); got != wantWidth {
			t.Errorf("table line %d display width = %d, want %d:\n%q", i, got, wantWidth, l)
		}
	}
}

func TestStringWidth(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"hello", 5},
		{"café", 4},
		{"café ☕", 7}, // 4 (café) + 1 (space) + 2 (☕)
		{"日本語", 6},
		{"🚀 rocket", 9}, // 2 (🚀) + 1 (space) + 6 (rocket)
		{"\x00abc", 3},  // null byte is width 0

		// Combining marks are zero terminal width: "e" + U+0301 renders as
		// one column, not two. The precomposed form (U+00E9) is one rune.
		{"é", 1},
		{"café", 4},
		{"ą", 1},
		{"x̨́", 1}, // multiple stacked marks stay one column
		{"é", 1},   // precomposed U+00E9 is a single width-1 rune

		// Halfwidth Katakana and Latin ligatures are one column each, unlike
		// their fullwidth counterparts.
		{"ｶﾀｶﾅ", 4},
		{"カタカナ", 8},
		{"find ﬁle", 8},
	}

	for _, tt := range tests {
		if got := stringWidth(tt.input); got != tt.want {
			t.Errorf("stringWidth(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

// TestPrettyOutputHalfwidthAlignment guards the reported bug: halfwidth
// Katakana (U+FF61-U+FFDC) and Latin ligatures (U+FB00-U+FB4F) each occupy one
// terminal column, so a message containing them must not shift its row's pipes
// out of alignment. Like TestPrettyOutputCombiningMarkAlignment, the check
// measures display width independently of stringWidth, the code under test.
func TestPrettyOutputHalfwidthAlignment(t *testing.T) {
	r := resultWith(
		Assumption{File: "katakana.go", Line: 4, Message: "if dog != nil { // ｶﾀｶﾅ"},
		Assumption{File: "ligature.go", Line: 7, Message: "if dog != nil { // find ﬁle"},
		Assumption{File: "plain.go", Line: 9, Message: "if value != nil { // pure ascii message longer"},
	)

	var buf bytes.Buffer
	if err := (PrettyOutput{}).Output(&buf, r); err != nil {
		t.Fatalf("Output: %v", err)
	}
	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	var tableLines []string
	for _, l := range lines {
		if l == "" {
			break
		}
		tableLines = append(tableLines, l)
	}
	if len(tableLines) != 7 {
		// 1 border + header + 1 separator + 3 rows + 1 border
		t.Fatalf("expected 7 table lines, got %d:\n%s", len(tableLines), out)
	}

	// Every rune in this input is exactly one terminal column, so the rune
	// count is the display width.
	visualWidth := func(s string) int {
		return len([]rune(s))
	}

	// The border is pure ASCII, so its byte length is its display width.
	wantWidth := len(tableLines[0])
	for i, l := range tableLines {
		if got := visualWidth(l); got != wantWidth {
			t.Errorf("table line %d display width = %d, want %d:\n%q", i, got, wantWidth, l)
		}
	}
}

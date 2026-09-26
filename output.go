package assumpgo

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"unicode"
)

// Output renders a Result.
type Output interface {
	Output(w io.Writer, result *Result) error
}

// PrettyOutput renders a human readable table, mirroring php-assumptions'
// pretty output. It owns the complete rendered document: the version banner
// (when a Version is set), the table, and the summary line.
type PrettyOutput struct {
	Version string
}

// NewPrettyOutput returns a PrettyOutput that prefixes its report with the
// "assumpgo analyser v<version> by quality-gates" banner.
func NewPrettyOutput(version string) *PrettyOutput {
	return &PrettyOutput{Version: version}
}

// Output writes the banner (when a Version is set), the table, and the summary
// line.
func (o PrettyOutput) Output(w io.Writer, result *Result) error {
	if o.Version != "" {
		if _, err := fmt.Fprintf(w, "assumpgo analyser v%s by quality-gates\n\n", o.Version); err != nil {
			return err
		}
	}

	if result.AssumptionsCount() > 0 {
		if err := writeTable(w, result.Assumptions()); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintf(
		w,
		"%d out of %d boolean expressions are assumptions (%d%%)\n",
		result.AssumptionsCount(),
		result.BoolExpressionsCount(),
		result.Percentage(),
	)

	return err
}

// escapeTerminalControls makes control characters visible in table cells so
// source text cannot change the terminal state or layout. Tabs retain the
// existing one-space rendering.
func escapeTerminalControls(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\t':
			b.WriteByte(' ')
		case unicode.IsControl(r):
			switch {
			case r <= 0xff:
				fmt.Fprintf(&b, `\x%02x`, r)
			case r <= 0xffff:
				fmt.Fprintf(&b, `\u%04x`, r)
			default:
				fmt.Fprintf(&b, `\U%08x`, r)
			}
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func writeTable(w io.Writer, assumptions []Assumption) error {
	headers := []string{"file", "line", "message"}
	rows := make([][]string, 0, len(assumptions))
	for _, a := range assumptions {
		rows = append(rows, []string{
			escapeTerminalControls(a.File),
			fmt.Sprintf("%d", a.Line),
			escapeTerminalControls(a.Message),
		})
	}

	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = stringWidth(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if w := stringWidth(cell); w > widths[i] {
				widths[i] = w
			}
		}
	}

	border := func(sep byte) string {
		var b strings.Builder
		total := 1
		for _, width := range widths {
			total += width + 3
		}
		for i := 0; i < total; i++ {
			b.WriteByte(sep)
		}
		return b.String()
	}

	writeRow := func(cells []string) error {
		var b strings.Builder
		b.WriteString("|")
		for i, cell := range cells {
			pad := widths[i] - stringWidth(cell)
			b.WriteString(" ")
			b.WriteString(cell)
			b.WriteString(strings.Repeat(" ", pad))
			b.WriteString(" |")
		}
		_, err := fmt.Fprintln(w, b.String())
		return err
	}

	if _, err := fmt.Fprintln(w, border('-')); err != nil {
		return err
	}
	if err := writeRow(headers); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, border('=')); err != nil {
		return err
	}
	for _, row := range rows {
		if err := writeRow(row); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w, border('-'))

	return err
}

// wideRanges are the East Asian Wide / Fullwidth code point ranges, listed
// individually rather than as the single span 0x2e80..0xffef they replace:
// that span also swallowed narrow blocks such as Alphabetic Presentation Forms
// (Latin ligatures like \ufb01) and the Halfwidth Forms at 0xff61..0xffdc,
// which are one column each.
var wideRanges = [...]struct{ lo, hi rune }{
	{0x1100, 0x11ff},    // Hangul Jamo
	{0x2600, 0x27bf},    // Misc Symbols and Dingbats
	{0x2e80, 0xa4cf},    // CJK radicals through Yi
	{0xac00, 0xd7af},    // Hangul syllables
	{0xf900, 0xfaff},    // CJK compatibility ideographs
	{0xfe10, 0xfe4f},    // vertical and CJK compatibility forms
	{0xff01, 0xff60},    // fullwidth ASCII forms
	{0xffe0, 0xffee},    // fullwidth signs
	{0x1f000, 0x10ffff}, // emoji and symbol planes
}

func runeWidth(r rune) int {
	if r == '\t' {
		return 1
	}
	if r < 32 || (r >= 0x7f && r < 0xa0) {
		return 0
	}
	// Combining marks and format characters (zero-width space, BOM) occupy
	// zero terminal columns.
	if unicode.In(r, unicode.Mn, unicode.Me, unicode.Cf) {
		return 0
	}
	// Regional indicators occupy one column each, so a flag pair occupies two.
	if r >= 0x1f1e6 && r <= 0x1f1ff {
		return 1
	}
	// U+2764 is narrow even though the enclosing Misc Symbols range is wide.
	if r == 0x2764 {
		return 1
	}
	for _, wr := range wideRanges {
		if r >= wr.lo && r <= wr.hi {
			return 2
		}
	}
	return 1
}

// stringWidth calculates the monospace visual display width of a string.
// Horizontal tabs and printable ASCII characters have width 1. Control,
// non-printable, combining, and format characters have width 0.
// East Asian Wide / Fullwidth characters and common emojis have width 2.
func stringWidth(s string) int {
	w := 0
	for _, r := range s {
		w += runeWidth(r)
	}
	return w
}

// XMLOutput renders a checkstyle-style XML report, mirroring php-assumptions'
// xml output so it can be consumed by CI tooling.
type XMLOutput struct{}

type checkstyle struct {
	XMLName xml.Name         `xml:"checkstyle"`
	Files   []checkstyleFile `xml:"file"`
}

type checkstyleFile struct {
	Name   string            `xml:"name,attr"`
	Errors []checkstyleError `xml:"error"`
}

type checkstyleError struct {
	Line     int    `xml:"line,attr"`
	Severity string `xml:"severity,attr"`
	Message  string `xml:"message,attr"`
	Source   string `xml:"source,attr"`
}

// Output writes the assumptions as checkstyle XML.
func (XMLOutput) Output(w io.Writer, result *Result) error {
	byFile := map[string]*checkstyleFile{}
	var order []string

	for _, a := range result.Assumptions() {
		f, ok := byFile[a.File]
		if !ok {
			f = &checkstyleFile{Name: a.File}
			byFile[a.File] = f
			order = append(order, a.File)
		}
		f.Errors = append(f.Errors, checkstyleError{
			Line:     a.Line,
			Severity: "error",
			Message:  a.Message,
			Source:   "assumpgo",
		})
	}

	doc := checkstyle{}
	for _, name := range order {
		doc.Files = append(doc.Files, *byFile[name])
	}

	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}

	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return err
	}

	_, err := io.WriteString(w, "\n")

	return err
}

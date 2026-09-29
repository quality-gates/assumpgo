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

// wideRanges are the code points libc wcwidth (macOS, en_US.UTF-8) reports
// as two columns, listed individually rather than as the coarse spans they
// replace: those also swallowed narrow blocks such as Alphabetic Presentation
// Forms (Latin ligatures like \ufb01), the Halfwidth Forms at 0xff61..0xffee,
// the Yijing hexagrams at 0x4dc0..0x4dff, and most of the SMP symbols, which
// are one column each. Nonspacing and spacing marks inside a range (such as
// 0x302a..0x302d) are caught as zero-width before the ranges are consulted.
// Regional indicators (0x1f1e6..0x1f1ff) fall between the ranges: they occupy
// one column each, so a flag pair occupies two.
var wideRanges = [...]struct{ lo, hi rune }{
	{0x1100, 0x115f},   // Hangul Jamo leading consonants
	{0x231a, 0x231b},   // watch, hourglass
	{0x2329, 0x232a},   // angle brackets
	{0x23e9, 0x23ec},   // double triangles
	{0x23f0, 0x23f0},   // alarm clock
	{0x23f3, 0x23f3},   // hourglass with flowing sand
	{0x25fd, 0x25fe},   // medium small squares
	{0x2614, 0x2615},   // umbrella with rain, hot beverage
	{0x2648, 0x2653},   // zodiac signs
	{0x267f, 0x267f},   // wheelchair
	{0x2693, 0x2693},   // anchor
	{0x26a1, 0x26a1},   // high voltage
	{0x26aa, 0x26ab},   // medium circles
	{0x26bd, 0x26be},   // soccer ball, baseball
	{0x26c4, 0x26c5},   // snowman without snow, sun behind cloud
	{0x26ce, 0x26ce},   // ophiuchus
	{0x26d4, 0x26d4},   // no entry
	{0x26ea, 0x26ea},   // church
	{0x26f2, 0x26f3},   // fountain, flag in hole
	{0x26f5, 0x26f5},   // sailboat
	{0x26fa, 0x26fa},   // tent
	{0x26fd, 0x26fd},   // fuel pump
	{0x2705, 0x2705},   // check mark button
	{0x270a, 0x270b},   // raised fist, raised hand
	{0x2728, 0x2728},   // sparkles
	{0x274c, 0x274c},   // cross mark
	{0x274e, 0x274e},   // cross mark button
	{0x2753, 0x2755},   // question and exclamation ornaments
	{0x2757, 0x2757},   // heavy exclamation mark
	{0x2795, 0x2797},   // heavy plus, minus, division
	{0x27b0, 0x27b0},   // curly loop
	{0x27bf, 0x27bf},   // double curly loop
	{0x2b1b, 0x2b1c},   // large squares
	{0x2b50, 0x2b50},   // star
	{0x2b55, 0x2b55},   // heavy large circle
	{0x2e80, 0x303e},   // CJK radicals through CJK symbols and punctuation
	{0x3041, 0x3247},   // Hiragana through circled ideographs
	{0x3250, 0x4dbf},   // enclosed CJK through CJK Extension A
	{0x4e00, 0xa4cf},   // CJK unified ideographs through Yi
	{0xa960, 0xa97c},   // Hangul Jamo Extended-A
	{0xac00, 0xd7af},   // Hangul syllables
	{0xf900, 0xfaff},   // CJK compatibility ideographs
	{0xfe10, 0xfe6b},   // vertical forms through small form variants
	{0xff01, 0xff60},   // fullwidth ASCII forms
	{0xffe0, 0xffe6},   // fullwidth signs
	{0x16fe0, 0x1b2fb}, // ideographic symbols, Tangut, Khitan, Kana supplements
	{0x1f004, 0x1f004}, // mahjong red dragon
	{0x1f0cf, 0x1f0cf}, // playing card black joker
	{0x1f18e, 0x1f18e}, // negative squared AB
	{0x1f191, 0x1f19a}, // squared CL through squared VS
	{0x1f200, 0x1f320}, // enclosed ideographic supplement through shooting star
	{0x1f32d, 0x1f335}, // hot dog through cactus
	{0x1f337, 0x1f37c}, // tulip through baby bottle
	{0x1f37e, 0x1f393}, // bottle with popping cork through graduation cap
	{0x1f3a0, 0x1f3ca}, // carousel horse through swimmer
	{0x1f3cf, 0x1f3d3}, // cricket through table tennis
	{0x1f3e0, 0x1f3f0}, // house through European castle
	{0x1f3f4, 0x1f3f4}, // waving black flag
	{0x1f3f8, 0x1f43e}, // badminton through paw prints
	{0x1f440, 0x1f440}, // eyes
	{0x1f442, 0x1f4fc}, // ear through videocassette
	{0x1f4ff, 0x1f53d}, // prayer beads through down-pointing small triangle
	{0x1f54b, 0x1f54e}, // kaaba through menorah
	{0x1f550, 0x1f567}, // clock faces
	{0x1f57a, 0x1f57a}, // man dancing
	{0x1f595, 0x1f596}, // middle finger, vulcan salute
	{0x1f5a4, 0x1f5a4}, // black heart
	{0x1f5fb, 0x1f64f}, // mount fuji through emoticons
	{0x1f680, 0x1f6c5}, // transport and map symbols
	{0x1f6cc, 0x1f6cc}, // sleeping accommodation
	{0x1f6d0, 0x1f6d2}, // place of worship through shopping trolley
	{0x1f6d5, 0x1f6df}, // hindu temple through ring buoy
	{0x1f6eb, 0x1f6ec}, // airplane departure, arrival
	{0x1f6f4, 0x1f6fc}, // scooter through roller skate
	{0x1f7e0, 0x1f7f0}, // large coloured circles and squares
	{0x1f90c, 0x1f93a}, // supplemental symbols and pictographs
	{0x1f93c, 0x1f945},
	{0x1f947, 0x1f9ff},
	{0x1fa70, 0x1faf8}, // symbols and pictographs extended-A
	{0x20000, 0x3fffd}, // supplementary and tertiary ideographic planes
}

func runeWidth(r rune) int {
	if r == '\t' {
		return 1
	}
	if r < 32 || (r >= 0x7f && r < 0xa0) {
		return 0
	}
	// Nonspacing and enclosing marks, format characters (zero-width space,
	// BOM), line and paragraph separators, and conjoining Hangul medial vowels
	// and final consonants occupy zero terminal columns.
	if unicode.In(r, unicode.Mn, unicode.Me, unicode.Cf, unicode.Zl, unicode.Zp) {
		return 0
	}
	if r >= 0x1160 && r <= 0x11ff {
		return 0
	}
	for _, wr := range wideRanges {
		if r >= wr.lo && r <= wr.hi {
			return 2
		}
	}
	// Spacing marks occupy zero columns too, except the few inside a wide
	// range (Hangul tone marks 0x302e..0x302f), which libc counts as wide.
	if unicode.Is(unicode.Mc, r) {
		return 0
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

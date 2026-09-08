package assumpgo

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// Output renders a Result.
type Output interface {
	Output(w io.Writer, result *Result) error
}

// PrettyOutput renders a human readable table, mirroring php-assumptions'
// pretty output.
type PrettyOutput struct{}

// Output writes the table and summary line.
func (PrettyOutput) Output(w io.Writer, result *Result) error {
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

func writeTable(w io.Writer, assumptions []Assumption) error {
	headers := []string{"file", "line", "message"}
	rows := make([][]string, 0, len(assumptions))
	for _, a := range assumptions {
		rows = append(rows, []string{a.File, fmt.Sprintf("%d", a.Line), a.Message})
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

func runeWidth(r rune) int {
	if r < 32 || (r >= 0x7f && r < 0xa0) {
		return 0
	}
	if (r >= 0x1100 && r <= 0x11ff) ||
		(r >= 0x2600 && r <= 0x27bf) ||
		(r >= 0x2e80 && r <= 0xffef) ||
		r >= 0x1f000 {
		return 2
	}
	return 1
}

// stringWidth calculates the monospace visual display width of a string.
// Printable ASCII characters have width 1. Control and non-printable characters have width 0.
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

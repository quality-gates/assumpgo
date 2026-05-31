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

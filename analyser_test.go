package assumpgo

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func analyseExample(t *testing.T) *Result {
	t.Helper()
	file := filepath.Join("testdata", "fixtures", "example.go")
	analyser := NewAnalyser(NewDetector(), nil)
	result, err := analyser.Analyse([]string{file})
	if err != nil {
		t.Fatalf("analyse: %v", err)
	}
	return result
}

func TestAnalyserDetectsAssumptions(t *testing.T) {
	file := filepath.Join("testdata", "fixtures", "example.go")
	result := analyseExample(t)

	want := []Assumption{
		{File: file, Line: 7, Message: "if test && len(bla) > 0 {"},
		{File: file, Line: 9, Message: "} else if !test {"},
		{File: file, Line: 13, Message: "for test {"},
		{File: file, Line: 17, Message: "for i := 0; i != 0; i++ {"},
	}

	if got := result.Assumptions(); !reflect.DeepEqual(got, want) {
		t.Errorf("assumptions mismatch:\n got: %#v\nwant: %#v", got, want)
	}
}

func TestAnalyserCountsBoolExpressions(t *testing.T) {
	result := analyseExample(t)

	if got := result.BoolExpressionsCount(); got != 7 {
		t.Errorf("BoolExpressionsCount() = %d, want 7", got)
	}
	if got := result.AssumptionsCount(); got != 4 {
		t.Errorf("AssumptionsCount() = %d, want 4", got)
	}
	if got := result.Percentage(); got != 57 {
		t.Errorf("Percentage() = %d, want 57", got)
	}
}

func TestAnalyserDetectsNilCheck(t *testing.T) {
	file := filepath.Join("testdata", "fixtures", "dog.go")
	analyser := NewAnalyser(NewDetector(), nil)
	result, err := analyser.Analyse([]string{file})
	if err != nil {
		t.Fatalf("analyse: %v", err)
	}

	if got := result.AssumptionsCount(); got != 1 {
		t.Fatalf("AssumptionsCount() = %d, want 1", got)
	}
	if msg := result.Assumptions()[0].Message; msg != "if dog != nil {" {
		t.Errorf("message = %q, want %q", msg, "if dog != nil {")
	}
	if got := result.BoolExpressionsCount(); got != 1 {
		t.Errorf("BoolExpressionsCount() = %d, want 1", got)
	}
	if got := result.Percentage(); got != 100 {
		t.Errorf("Percentage() = %d, want 100", got)
	}
}

func TestAnalyserFindsNoAssumptionsInAssertion(t *testing.T) {
	file := filepath.Join("testdata", "fixtures", "cat.go")
	analyser := NewAnalyser(NewDetector(), nil)
	result, err := analyser.Analyse([]string{file})
	if err != nil {
		t.Fatalf("analyse: %v", err)
	}

	if got := result.AssumptionsCount(); got != 0 {
		t.Errorf("AssumptionsCount() = %d, want 0", got)
	}
}

func TestAnalyserHonoursExcludes(t *testing.T) {
	file := filepath.Join("testdata", "fixtures", "dog.go")
	analyser := NewAnalyser(NewDetector(), []string{file})
	result, err := analyser.Analyse([]string{file})
	if err != nil {
		t.Fatalf("analyse: %v", err)
	}

	if got := result.AssumptionsCount(); got != 0 {
		t.Errorf("excluded file still analysed: count = %d", got)
	}
}

func TestPercentageZeroWhenNoBoolExpressions(t *testing.T) {
	r := &Result{}
	if got := r.Percentage(); got != 0 {
		t.Errorf("Percentage() = %d, want 0", got)
	}

	// Even with assumptions present, a zero boolean-expression count must
	// short-circuit to 0 rather than divide by zero.
	r.addAssumption("a.go", 1, "x != nil")
	if got := r.Percentage(); got != 0 {
		t.Errorf("Percentage() with no bool expressions = %d, want 0", got)
	}
}

func TestPercentageRounds(t *testing.T) {
	r := &Result{boolExpressionsCount: 8}
	r.addAssumption("a.go", 1, "x")
	// 1/8 = 12.5% -> rounds to 13 (verifies the *100 scaling and rounding).
	if got := r.Percentage(); got != 13 {
		t.Errorf("Percentage() = %d, want 13", got)
	}
}

func TestReadLine(t *testing.T) {
	lines := []string{"  first  ", "second", "third"}
	cases := map[int]string{
		1: "first", // trimmed
		2: "second",
		3: "third",
		0: "", // below range
		4: "", // above range
	}
	for line, want := range cases {
		if got := readLine(lines, line); got != want {
			t.Errorf("readLine(_, %d) = %q, want %q", line, got, want)
		}
	}
	if got := readLine(nil, 1); got != "" {
		t.Errorf("readLine(nil, 1) = %q, want empty", got)
	}
}

func TestAnalyseReturnsErrorForMissingFile(t *testing.T) {
	analyser := NewAnalyser(NewDetector(), nil)
	if _, err := analyser.Analyse([]string{"does-not-exist.go"}); err == nil {
		t.Error("expected an error for a missing file")
	}
}

func TestAnalyseReturnsErrorForInvalidGo(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.go")
	if err := os.WriteFile(bad, []byte("package x\nfunc {"), 0o644); err != nil {
		t.Fatal(err)
	}
	analyser := NewAnalyser(NewDetector(), nil)
	if _, err := analyser.Analyse([]string{bad}); err == nil {
		t.Error("expected a parse error for invalid Go")
	}
}

// TestAnalyseExcludeContinues ensures an excluded file is skipped without
// halting analysis of the files that follow it.
func TestAnalyseExcludeContinues(t *testing.T) {
	cat := filepath.Join("testdata", "fixtures", "cat.go")
	dog := filepath.Join("testdata", "fixtures", "dog.go")

	analyser := NewAnalyser(NewDetector(), []string{cat})
	result, err := analyser.Analyse([]string{cat, dog})
	if err != nil {
		t.Fatalf("analyse: %v", err)
	}

	if got := result.AssumptionsCount(); got != 1 {
		t.Errorf("AssumptionsCount() = %d, want 1 (dog.go must still be analysed)", got)
	}
}

package assumpgo

import (
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
}

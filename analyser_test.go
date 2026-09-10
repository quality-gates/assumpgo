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

	// 5 statement/logic contexts (2 if + else-if, 2 for, 1 &&) plus the 2
	// assumption nodes Scan flags outside those contexts (!test, i != 0)
	// which must also count toward the denominator (issue #34).
	if got := result.BoolExpressionsCount(); got != 9 {
		t.Errorf("BoolExpressionsCount() = %d, want 9", got)
	}
	if got := result.AssumptionsCount(); got != 4 {
		t.Errorf("AssumptionsCount() = %d, want 4", got)
	}
	if got := result.Percentage(); got != 44 {
		t.Errorf("Percentage() = %d, want 44", got)
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
	// The `if` and the `!=` comparison each count as a boolean expression.
	if got := result.BoolExpressionsCount(); got != 2 {
		t.Errorf("BoolExpressionsCount() = %d, want 2", got)
	}
	if got := result.Percentage(); got != 50 {
		t.Errorf("Percentage() = %d, want 50", got)
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

func TestAnalyserIgnoresNamedBooleanConstants(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "const.go")
	code := `package p

const enabled = true

func check() bool {
	if enabled {
		return true
	}
	return false
}
`
	if err := os.WriteFile(src, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}

	analyser := NewAnalyser(NewDetector(), nil)
	result, err := analyser.Analyse([]string{src})
	if err != nil {
		t.Fatalf("analyse: %v", err)
	}

	if got := result.AssumptionsCount(); got != 0 {
		t.Errorf("AssumptionsCount() = %d, want 0; assumptions: %#v", got, result.Assumptions())
	}
	if got := result.BoolExpressionsCount(); got != 1 {
		t.Errorf("BoolExpressionsCount() = %d, want 1", got)
	}
	if got := result.Percentage(); got != 0 {
		t.Errorf("Percentage() = %d, want 0", got)
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

func TestAnalyserHonoursExcludesPathSyntax(t *testing.T) {
	rel := filepath.Join("testdata", "fixtures", "dog.go")
	abs, err := filepath.Abs(rel)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		exclude string
		target  string
	}{
		{
			name:    "exclude with leading dot-slash",
			exclude: "./testdata/fixtures/dog.go",
			target:  "testdata/fixtures/dog.go",
		},
		{
			name:    "target with leading dot-slash",
			exclude: "testdata/fixtures/dog.go",
			target:  "./testdata/fixtures/dog.go",
		},
		{
			name:    "exclude with redundant separators",
			exclude: "testdata//fixtures/dog.go",
			target:  "testdata/fixtures/dog.go",
		},
		{
			name:    "target with redundant separators",
			exclude: "testdata/fixtures/dog.go",
			target:  "testdata//fixtures/dog.go",
		},
		{
			name:    "exclude with relative dot segment",
			exclude: "testdata/fixtures/../fixtures/dog.go",
			target:  "testdata/fixtures/dog.go",
		},
		{
			name:    "exclude relative, target absolute",
			exclude: rel,
			target:  abs,
		},
		{
			name:    "exclude absolute, target relative",
			exclude: abs,
			target:  rel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyser := NewAnalyser(NewDetector(), []string{tt.exclude})
			result, err := analyser.Analyse([]string{tt.target})
			if err != nil {
				t.Fatalf("analyse: %v", err)
			}
			if got := result.AssumptionsCount(); got != 0 {
				t.Errorf("file %q was not excluded by %q (got %d assumptions, want 0)", tt.target, tt.exclude, got)
			}
		})
	}
}

func TestAnalyserExcludeDifferentFileAbsoluteDoesNotExclude(t *testing.T) {
	dog := filepath.Join("testdata", "fixtures", "dog.go")
	cat := filepath.Join("testdata", "fixtures", "cat.go")
	absCat, err := filepath.Abs(cat)
	if err != nil {
		t.Fatal(err)
	}

	analyser := NewAnalyser(NewDetector(), []string{absCat})
	result, err := analyser.Analyse([]string{dog})
	if err != nil {
		t.Fatalf("analyse: %v", err)
	}
	if got := result.AssumptionsCount(); got != 1 {
		t.Errorf("dog.go was excluded by absolute cat.go (got %d assumptions, want 1)", got)
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

// TestPercentageCoversAssumptionNodes is the regression test for issue #34:
// `Scan` flags `!=` and `!var` anywhere in the tree, so those nodes must also
// contribute to the boolean-expression denominator. Otherwise the ratio can be
// 0% with findings present (a top-level `return x != nil`) or exceed 100%
// (`return x != nil && y != nil` counts two assumptions against only the `&&`).
func TestPercentageCoversAssumptionNodes(t *testing.T) {
	tests := []struct {
		name            string
		code            string
		wantAssumptions int
		wantBoolExprs   int
		wantPercentage  int
	}{
		{
			name: "top-level not-equal is its own boolean expression",
			code: `package main

func F(x *int) bool {
	return x != nil
}
`,
			wantAssumptions: 1,
			wantBoolExprs:   1,
			wantPercentage:  100,
		},
		{
			name: "and of two not-equals counts each side in the denominator",
			code: `package main

func F(x, y *int) bool {
	return x != nil && y != nil
}
`,
			wantAssumptions: 2,
			wantBoolExprs:   3,
			wantPercentage:  67,
		},
		{
			name: "top-level boolean-not is its own boolean expression",
			code: `package main

func F(ready bool) bool {
	return !ready
}
`,
			wantAssumptions: 1,
			wantBoolExprs:   1,
			wantPercentage:  100,
		},
		{
			name: "strict equality is neither assumption nor boolean expression",
			code: `package main

func F(x *int) bool {
	return x == nil
}
`,
			wantAssumptions: 0,
			wantBoolExprs:   0,
			wantPercentage:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			src := filepath.Join(dir, "repro.go")
			if err := os.WriteFile(src, []byte(tt.code), 0o644); err != nil {
				t.Fatal(err)
			}
			analyser := NewAnalyser(NewDetector(), nil)
			result, err := analyser.Analyse([]string{src})
			if err != nil {
				t.Fatalf("analyse: %v", err)
			}

			if got := result.AssumptionsCount(); got != tt.wantAssumptions {
				t.Errorf("AssumptionsCount() = %d, want %d; assumptions: %#v", got, tt.wantAssumptions, result.Assumptions())
			}
			if got := result.BoolExpressionsCount(); got != tt.wantBoolExprs {
				t.Errorf("BoolExpressionsCount() = %d, want %d", got, tt.wantBoolExprs)
			}
			if got := result.Percentage(); got != tt.wantPercentage {
				t.Errorf("Percentage() = %d, want %d", got, tt.wantPercentage)
			}
		})
	}
}

// TestAnalyserDetectsLeftAssociativeVarComparisonMix is the regression for
// issue #39: extra bare variables to the left of a mix must not hide it.
func TestAnalyserDetectsLeftAssociativeVarComparisonMix(t *testing.T) {
	tests := []struct {
		name            string
		code            string
		wantAssumptions int
		wantBoolExprs   int
		wantMessage     string
	}{
		{
			name: "vars to the left of a comparison",
			code: `package p

func F(x, y bool, n int) {
	if x && y && n == 1 {
	}
}
`,
			wantAssumptions: 1,
			wantBoolExprs:   3,
			wantMessage:     "if x && y && n == 1 {",
		},
		{
			name: "parenthesized vars to the left of a comparison",
			code: `package p

func F(x, y bool, n int) {
	if (x && y) && n == 1 {
	}
}
`,
			wantAssumptions: 1,
			wantBoolExprs:   3,
			wantMessage:     "if (x && y) && n == 1 {",
		},
		{
			name: "pure var chain is still not a mix",
			code: `package p

func F(x, y, z bool) {
	if x && y && z {
	}
}
`,
			wantAssumptions: 0,
			wantBoolExprs:   3,
		},
		{
			name: "two comparisons are still not a mix",
			code: `package p

func F(x, y int) {
	if x == 1 && y == 2 {
	}
}
`,
			wantAssumptions: 0,
			wantBoolExprs:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			src := filepath.Join(dir, "mix.go")
			if err := os.WriteFile(src, []byte(tt.code), 0o644); err != nil {
				t.Fatal(err)
			}
			analyser := NewAnalyser(NewDetector(), nil)
			result, err := analyser.Analyse([]string{src})
			if err != nil {
				t.Fatalf("analyse: %v", err)
			}

			if got := result.AssumptionsCount(); got != tt.wantAssumptions {
				t.Errorf("AssumptionsCount() = %d, want %d; assumptions: %#v", got, tt.wantAssumptions, result.Assumptions())
			}
			if got := result.BoolExpressionsCount(); got != tt.wantBoolExprs {
				t.Errorf("BoolExpressionsCount() = %d, want %d", got, tt.wantBoolExprs)
			}
			if tt.wantAssumptions > 0 {
				if msg := result.Assumptions()[0].Message; msg != tt.wantMessage {
					t.Errorf("message = %q, want %q", msg, tt.wantMessage)
				}
			}
		})
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

func TestAnalyserFindsNoAssumptionsInInvertedCommaOk(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "guard.go")
	code := `package main

type Dog struct{}

func Check(cat any) {
	if _, ok := cat.(*Dog); !ok {
		return
	}
}
`
	if err := os.WriteFile(src, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	analyser := NewAnalyser(NewDetector(), nil)
	result, err := analyser.Analyse([]string{src})
	if err != nil {
		t.Fatalf("analyse: %v", err)
	}

	if got := result.AssumptionsCount(); got != 0 {
		t.Errorf("AssumptionsCount() = %d, want 0; assumptions: %#v", got, result.Assumptions())
	}
	if got := result.BoolExpressionsCount(); got != 1 {
		t.Errorf("BoolExpressionsCount() = %d, want 1", got)
	}
	if got := result.Percentage(); got != 0 {
		t.Errorf("Percentage() = %d, want 0", got)
	}
}

func TestAnalyserDoesNotFlagCommaOkVariableInLogicalCondition(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "guard.go")
	code := `package main

func Check(x any) {
	if v, ok := x.(*int); ok && v != nil {
		_ = v
	}
}
`
	if err := os.WriteFile(src, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}

	analyser := NewAnalyser(NewDetector(), nil)
	result, err := analyser.Analyse([]string{src})
	if err != nil {
		t.Fatalf("analyse: %v", err)
	}
	want := []Assumption{{File: src, Line: 4, Message: "if v, ok := x.(*int); ok && v != nil {"}}
	if got := result.Assumptions(); !reflect.DeepEqual(got, want) {
		t.Errorf("assumptions mismatch:\n got: %#v\nwant: %#v", got, want)
	}
	if got := result.BoolExpressionsCount(); got != 3 {
		t.Errorf("BoolExpressionsCount() = %d, want 3", got)
	}
	if got := result.Percentage(); got != 33 {
		t.Errorf("Percentage() = %d, want 33", got)
	}
}

func TestAnalyserHandlesCommaOkLogicalConditions(t *testing.T) {
	tests := []struct {
		name            string
		code            string
		wantLine        int
		wantMessage     string
		wantAssumptions int
		wantBoolExprs   int
		wantPercentage  int
	}{
		{
			name: "map lookup with reversed or",
			code: `package main

func Check(m map[string]string, k string) {
	if v, ok := m[k]; v != "" || ok {
		_ = v
	}
}
`,
			wantLine:        4,
			wantMessage:     `if v, ok := m[k]; v != "" || ok {`,
			wantAssumptions: 1,
			wantBoolExprs:   3,
			wantPercentage:  33,
		},
		{
			name: "inverted type assertion with and",
			code: `package main

func Check(x any) {
	if v, ok := x.(*int); !ok && v != nil {
		_ = v
	}
}
`,
			wantLine:        4,
			wantMessage:     `if v, ok := x.(*int); !ok && v != nil {`,
			wantAssumptions: 1,
			wantBoolExprs:   3,
			wantPercentage:  33,
		},
		{
			name: "for condition",
			code: `package main

func Check(m map[string]string, k string) {
	for v, ok := m[k]; ok || v != ""; {
		break
	}
}
`,
			wantLine:        4,
			wantMessage:     `for v, ok := m[k]; ok || v != ""; {`,
			wantAssumptions: 1,
			wantBoolExprs:   3,
			wantPercentage:  33,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			src := filepath.Join(dir, "guard.go")
			if err := os.WriteFile(src, []byte(tt.code), 0o644); err != nil {
				t.Fatal(err)
			}

			analyser := NewAnalyser(NewDetector(), nil)
			result, err := analyser.Analyse([]string{src})
			if err != nil {
				t.Fatalf("analyse: %v", err)
			}

			want := []Assumption{{File: src, Line: tt.wantLine, Message: tt.wantMessage}}
			if got := result.Assumptions(); !reflect.DeepEqual(got, want) {
				t.Errorf("assumptions mismatch:\n got: %#v\nwant: %#v", got, want)
			}
			if got := result.BoolExpressionsCount(); got != tt.wantBoolExprs {
				t.Errorf("BoolExpressionsCount() = %d, want %d", got, tt.wantBoolExprs)
			}
			if got := result.AssumptionsCount(); got != tt.wantAssumptions {
				t.Errorf("AssumptionsCount() = %d, want %d", got, tt.wantAssumptions)
			}
			if got := result.Percentage(); got != tt.wantPercentage {
				t.Errorf("Percentage() = %d, want %d", got, tt.wantPercentage)
			}
		})
	}
}

func TestAnalyserKeepsNonCommaOkLogicalAssumptions(t *testing.T) {
	tests := []struct {
		name            string
		code            string
		wantLine        int
		wantMessage     string
		wantAssumptions int
	}{
		{
			name: "function call init",
			code: `package main

func lookup() (int, bool) { return 1, true }

func Check() {
	if v, ok := lookup(); ok && v != 0 {
		_ = v
	}
}
`,
			wantLine:        6,
			wantMessage:     `if v, ok := lookup(); ok && v != 0 {`,
			wantAssumptions: 2,
		},
		{
			name: "bare variable without init",
			code: `package main

func Check(ready bool, v *int) {
	if ready && v != nil {
		_ = v
	}
}
`,
			wantLine:        4,
			wantMessage:     `if ready && v != nil {`,
			wantAssumptions: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			src := filepath.Join(dir, "guard.go")
			if err := os.WriteFile(src, []byte(tt.code), 0o644); err != nil {
				t.Fatal(err)
			}

			analyser := NewAnalyser(NewDetector(), nil)
			result, err := analyser.Analyse([]string{src})
			if err != nil {
				t.Fatalf("analyse: %v", err)
			}

			want := []Assumption{
				{File: src, Line: tt.wantLine, Message: tt.wantMessage},
				{File: src, Line: tt.wantLine, Message: tt.wantMessage},
			}
			if got := result.Assumptions(); !reflect.DeepEqual(got, want) {
				t.Errorf("assumptions mismatch:\n got: %#v\nwant: %#v", got, want)
			}
			if got := result.AssumptionsCount(); got != tt.wantAssumptions {
				t.Errorf("AssumptionsCount() = %d, want %d", got, tt.wantAssumptions)
			}
		})
	}
}

func TestAnalyserFindsNoAssumptionsInInvertedMapLookup(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "map.go")
	code := `package main

func Lookup(m map[string]int, k string) int {
	if val, ok := m[k]; !ok {
		return 0
	} else {
		return val
	}
}
`
	if err := os.WriteFile(src, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	analyser := NewAnalyser(NewDetector(), nil)
	result, err := analyser.Analyse([]string{src})
	if err != nil {
		t.Fatalf("analyse: %v", err)
	}

	if got := result.AssumptionsCount(); got != 0 {
		t.Errorf("AssumptionsCount() = %d, want 0; assumptions: %#v", got, result.Assumptions())
	}
	if got := result.BoolExpressionsCount(); got != 1 {
		t.Errorf("BoolExpressionsCount() = %d, want 1", got)
	}
}

func TestAnalyserFindsNoAssumptionsInForInvertedCommaOk(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "loop.go")
	code := `package main

func Loop(m map[string]int, k string) {
	for _, ok := m[k]; !ok; {
		break
	}
}
`
	if err := os.WriteFile(src, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	analyser := NewAnalyser(NewDetector(), nil)
	result, err := analyser.Analyse([]string{src})
	if err != nil {
		t.Fatalf("analyse: %v", err)
	}

	if got := result.AssumptionsCount(); got != 0 {
		t.Errorf("AssumptionsCount() = %d, want 0; assumptions: %#v", got, result.Assumptions())
	}
}

func TestAnalyserFlagsGeneralBooleanNotOutsideCommaOk(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "not.go")
	code := `package main

func Guard(ready bool) {
	if !ready {
		return
	}
}
`
	if err := os.WriteFile(src, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	analyser := NewAnalyser(NewDetector(), nil)
	result, err := analyser.Analyse([]string{src})
	if err != nil {
		t.Fatalf("analyse: %v", err)
	}

	if got := result.AssumptionsCount(); got != 1 {
		t.Fatalf("AssumptionsCount() = %d, want 1", got)
	}
}

func TestAnalyserFlagsBooleanNotWhenVariableDiffers(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "diff.go")
	code := `package main

func Guard(m map[string]int, k string, ready bool) {
	if val, ok := m[k]; !ready {
		_ = val
		return
	}
}
`
	if err := os.WriteFile(src, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	analyser := NewAnalyser(NewDetector(), nil)
	result, err := analyser.Analyse([]string{src})
	if err != nil {
		t.Fatalf("analyse: %v", err)
	}

	if got := result.AssumptionsCount(); got != 1 {
		t.Fatalf("AssumptionsCount() = %d, want 1", got)
	}
}

func TestAnalyserFlagsBooleanNotWhenInitIsNotCommaOk(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "call.go")
	code := `package main

func isReady() bool { return false }

func Guard() {
	if ready := isReady(); !ready {
		return
	}
}
`
	if err := os.WriteFile(src, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	analyser := NewAnalyser(NewDetector(), nil)
	result, err := analyser.Analyse([]string{src})
	if err != nil {
		t.Fatalf("analyse: %v", err)
	}

	if got := result.AssumptionsCount(); got != 1 {
		t.Fatalf("AssumptionsCount() = %d, want 1", got)
	}
}

func TestAnalyserDetectsBareVariableWithUnrelatedInitInIfAndFor(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "main.go")
	code := `package main

func doSomething() {}

func Check(running bool) {
	for i := 0; running; i++ {
	}
	if doSomething(); running {
	}
}
`
	if err := os.WriteFile(src, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	analyser := NewAnalyser(NewDetector(), nil)
	result, err := analyser.Analyse([]string{src})
	if err != nil {
		t.Fatalf("analyse: %v", err)
	}

	if got := result.BoolExpressionsCount(); got != 2 {
		t.Errorf("BoolExpressionsCount() = %d, want 2", got)
	}
	if got := result.AssumptionsCount(); got != 2 {
		t.Errorf("AssumptionsCount() = %d, want 2; assumptions: %#v", got, result.Assumptions())
	}
}

package assumpgo

import (
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"os"
	"strings"
)

// Assumption is a single weak boolean check found in the analysed source.
type Assumption struct {
	File    string
	Line    int
	Message string
}

// Result holds the outcome of an analysis run.
type Result struct {
	assumptions          []Assumption
	boolExpressionsCount int
}

func (r *Result) addAssumption(file string, line int, message string) {
	r.assumptions = append(r.assumptions, Assumption{File: file, Line: line, Message: message})
}

func (r *Result) increaseBoolExpressionsCount() {
	r.boolExpressionsCount++
}

// Assumptions returns every assumption found, in source order.
func (r *Result) Assumptions() []Assumption {
	return r.assumptions
}

// AssumptionsCount returns the number of assumptions found.
func (r *Result) AssumptionsCount() int {
	return len(r.assumptions)
}

// BoolExpressionsCount returns the number of boolean expressions analysed.
func (r *Result) BoolExpressionsCount() int {
	return r.boolExpressionsCount
}

// Percentage returns the rounded percentage of boolean expressions that are
// assumptions.
func (r *Result) Percentage() int {
	if r.boolExpressionsCount == 0 {
		return 0
	}

	return int(math.Round(float64(r.AssumptionsCount()) / float64(r.boolExpressionsCount) * 100))
}

// Analyser walks Go source files and records assumptions.
type Analyser struct {
	detector *Detector
	excludes map[string]struct{}
}

// NewAnalyser returns an Analyser. Files whose paths appear in excludes are
// skipped.
func NewAnalyser(detector *Detector, excludes []string) *Analyser {
	set := make(map[string]struct{}, len(excludes))
	for _, e := range excludes {
		set[e] = struct{}{}
	}

	return &Analyser{detector: detector, excludes: set}
}

// Analyse parses and inspects each file, returning the aggregated Result.
func (a *Analyser) Analyse(files []string) (*Result, error) {
	result := &Result{}

	for _, file := range files {
		if _, excluded := a.excludes[file]; excluded {
			continue
		}

		if err := a.analyseFile(file, result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (a *Analyser) analyseFile(path string, result *Result) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.SkipObjectResolution)
	if err != nil {
		return err
	}

	lines := strings.Split(string(src), "\n")

	ast.Inspect(f, func(node ast.Node) bool {
		if a.detector.IsBoolExpression(node) {
			result.increaseBoolExpressionsCount()
		}

		if a.detector.Scan(node) {
			line := fset.Position(node.Pos()).Line
			result.addAssumption(path, line, readLine(lines, line))
		}

		return true
	})

	return nil
}

func readLine(lines []string, line int) string {
	if line < 1 || line > len(lines) {
		return ""
	}

	return strings.TrimSpace(lines[line-1])
}

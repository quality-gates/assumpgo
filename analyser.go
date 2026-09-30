package assumpgo

import (
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"os"
	"path/filepath"
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

func identityPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return abs
}

// NewAnalyser returns an Analyser. Files whose paths appear in excludes are
// skipped.
func NewAnalyser(detector *Detector, excludes []string) *Analyser {
	set := make(map[string]struct{}, len(excludes))
	for _, e := range excludes {
		set[identityPath(e)] = struct{}{}
	}

	return &Analyser{detector: detector, excludes: set}
}

// Analyse parses and inspects each file, returning the aggregated Result.
func (a *Analyser) Analyse(files []string) (*Result, error) {
	resolver := NewPackageResolver()
	var sourceFiles []parsedSource
	var analysed []os.FileInfo

	for _, file := range files {
		clean := filepath.Clean(file)
		if a.isExcluded(clean) {
			continue
		}

		info, statErr := os.Stat(clean)
		if statErr == nil {
			if containsSameFile(analysed, info) {
				continue
			}
		}

		source, err := readParsedSource(clean)
		if err != nil {
			return nil, err
		}
		if statErr == nil {
			analysed = append(analysed, info)
		}
		sourceFiles = append(sourceFiles, source)
		resolver.addParsedFile(clean, source.file)
	}

	result := &Result{}
	for _, source := range sourceFiles {
		if err := a.analyseFile(source, result, resolver); err != nil {
			return nil, err
		}
	}
	return result, nil
}

type parsedSource struct {
	path string
	src  []byte
	fset *token.FileSet
	file *ast.File
}

func readParsedSource(path string) (parsedSource, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return parsedSource{}, err
	}

	fset := token.NewFileSet()
	// Keep parser object resolution enabled for declarations within this file.
	f, err := parser.ParseFile(fset, path, src, 0)
	if err != nil {
		return parsedSource{}, err
	}

	return parsedSource{path: path, src: src, fset: fset, file: f}, nil
}

func (a *Analyser) isExcluded(path string) bool {
	if _, excluded := a.excludes[identityPath(path)]; excluded {
		return true
	}

	for exclude := range a.excludes {
		if sameFile(exclude, path) {
			return true
		}
	}

	return false
}

func (a *Analyser) analyseFile(source parsedSource, result *Result, resolver packageConstLookup) error {
	path := source.path
	fset := source.fset
	f := source.file
	lines := strings.Split(string(source.src), "\n")
	scope := identifierScope{resolver: resolver, filePath: path, pkgName: f.Name.Name}

	ignored := make(map[ast.Node]struct{})
	ignoredAssumptions := make(map[ast.Node]struct{})

	ast.Inspect(f, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.IfStmt:
			if cond := a.detector.invertedCommaOkCond(n.Init, n.Cond); cond != nil {
				ignored[cond] = struct{}{}
				ignored[n.Cond] = struct{}{}
			}
			markCommaOkConditionNodes(n.Init, n.Cond, ignored, ignoredAssumptions, scope)
		case *ast.ForStmt:
			if cond := a.detector.invertedCommaOkCond(n.Init, n.Cond); cond != nil {
				ignored[cond] = struct{}{}
				ignored[n.Cond] = struct{}{}
			}
			markCommaOkConditionNodes(n.Init, n.Cond, ignored, ignoredAssumptions, scope)
		}

		if _, skip := ignored[node]; skip {
			return true
		}

		if a.detector.isBoolExpression(node, scope) {
			result.increaseBoolExpressionsCount()
		}

		if _, skip := ignoredAssumptions[node]; skip {
			return true
		}

		if a.detector.scan(node, scope) {
			// Ignore //line directives: the message is read from this
			// file, so the line must be physical (issue #87).
			line := fset.PositionFor(node.Pos(), false).Line
			result.addAssumption(path, line, readLine(lines, line))
			markNestedLogicalAssumptions(node, ignoredAssumptions)
		}

		return true
	})

	return nil
}

// markNestedLogicalAssumptions ignores nested && / || nodes under a mix.
// Record the chain as one assumption. Operand order does not change the
// count (issue #77). Do not descend past non-logical boundaries such as
// function calls or closures (issue #81).
func markNestedLogicalAssumptions(node ast.Node, ignoredAssumptions map[ast.Node]struct{}) {
	binary, ok := node.(*ast.BinaryExpr)
	if !ok || (binary.Op != token.LAND && binary.Op != token.LOR) {
		return
	}

	var mark func(n ast.Node)
	mark = func(n ast.Node) {
		inner, ok := unwrap(n).(*ast.BinaryExpr)
		if !ok || (inner.Op != token.LAND && inner.Op != token.LOR) {
			return
		}
		ignoredAssumptions[inner] = struct{}{}
		mark(inner.X)
		mark(inner.Y)
	}
	mark(binary.X)
	mark(binary.Y)
}

// markCommaOkConditionNodes traverses the condition expression to exempt
// the comma-ok ok / !ok guard and its logical mixes with other conditions.
// It stops traversal at non-logical boundaries (calls, composite literals,
// closures, and index expressions) so assumptions nested within those
// expressions are not suppressed.
func markCommaOkConditionNodes(init ast.Stmt, cond ast.Expr, ignored, ignoredAssumptions map[ast.Node]struct{}, scope identifierScope) {
	okName := commaOkVarName(init)
	if okName == "" || cond == nil {
		return
	}

	ast.Inspect(cond, func(node ast.Node) bool {
		switch node.(type) {
		case *ast.CallExpr, *ast.CompositeLit, *ast.FuncLit, *ast.IndexExpr:
			return false
		}
		if isCommaOkNotNode(node, okName) {
			ignored[node] = struct{}{}
		}
		if isCommaOkLogicalNodeInScope(node, okName, scope) {
			ignoredAssumptions[node] = struct{}{}
		}
		return true
	})
}

func readLine(lines []string, line int) string {
	if line < 1 || line > len(lines) {
		return ""
	}

	return strings.ReplaceAll(strings.TrimSpace(lines[line-1]), "\t", " ")
}

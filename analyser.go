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
	result := &Result{}
	consts := newConstIndex()

	for _, file := range files {
		clean := filepath.Clean(file)
		if a.isExcluded(clean) {
			continue
		}

		if err := a.analyseFile(clean, result, consts); err != nil {
			return nil, err
		}
	}

	return result, nil
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

func (a *Analyser) analyseFile(path string, result *Result, consts *constIndex) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	fset := token.NewFileSet()
	// Keep object resolution enabled so the detector can distinguish named
	// constants from variables.
	f, err := parser.ParseFile(fset, path, src, 0)
	if err != nil {
		return err
	}

	// Object resolution is per-file, so a constant declared in another file of
	// the same package is left unresolved and would look like a variable
	// (issue #58). Resolve those against the package's other files.
	resolvePackageConsts(f, consts.names(filepath.Dir(path), f.Name.Name))

	lines := strings.Split(string(src), "\n")

	ignored := make(map[ast.Node]struct{})
	ignoredAssumptions := make(map[ast.Node]struct{})

	ast.Inspect(f, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.IfStmt:
			if cond := a.detector.invertedCommaOkCond(n.Init, n.Cond); cond != nil {
				ignored[cond] = struct{}{}
				ignored[n.Cond] = struct{}{}
			}
			markCommaOkConditionNodes(n.Init, n.Cond, ignored, ignoredAssumptions)
		case *ast.ForStmt:
			if cond := a.detector.invertedCommaOkCond(n.Init, n.Cond); cond != nil {
				ignored[cond] = struct{}{}
				ignored[n.Cond] = struct{}{}
			}
			markCommaOkConditionNodes(n.Init, n.Cond, ignored, ignoredAssumptions)
		}

		if _, skip := ignored[node]; skip {
			return true
		}

		if a.detector.IsBoolExpression(node) {
			result.increaseBoolExpressionsCount()
		}

		if _, skip := ignoredAssumptions[node]; skip {
			return true
		}

		if a.detector.Scan(node) {
			line := fset.Position(node.Pos()).Line
			result.addAssumption(path, line, readLine(lines, line))
		}

		return true
	})

	return nil
}

func markCommaOkConditionNodes(init ast.Stmt, cond ast.Expr, ignored, ignoredAssumptions map[ast.Node]struct{}) {
	okName := commaOkVarName(init)
	if okName == "" || cond == nil {
		return
	}

	ast.Inspect(cond, func(node ast.Node) bool {
		if isCommaOkNotNode(node, okName) {
			ignored[node] = struct{}{}
		}
		if isCommaOkLogicalNode(node, okName) {
			ignoredAssumptions[node] = struct{}{}
		}
		return true
	})
}

func readLine(lines []string, line int) string {
	if line < 1 || line > len(lines) {
		return ""
	}

	return strings.TrimSpace(lines[line-1])
}

// constIndex caches the package-level constant names declared in a directory,
// keyed by directory and then by package name. A directory can hold more than
// one package (a `_test` external test package alongside the package proper),
// and a constant is only visible to files in its own package.
type constIndex struct {
	dirs map[string]map[string]map[string]struct{}
}

func newConstIndex() *constIndex {
	return &constIndex{dirs: make(map[string]map[string]map[string]struct{})}
}

// names returns the package-level constant names declared by any Go file in
// dir that belongs to package pkg.
func (c *constIndex) names(dir, pkg string) map[string]struct{} {
	byPkg, scanned := c.dirs[dir]
	if !scanned {
		byPkg = scanDirConsts(dir)
		c.dirs[dir] = byPkg
	}

	return byPkg[pkg]
}

// scanDirConsts parses every Go file in dir and groups the package-level
// constant names it declares by package name. Files that cannot be read or
// parsed contribute nothing rather than failing the run: they are context for
// the files actually being analysed, not targets themselves.
func scanDirConsts(dir string) map[string]map[string]struct{} {
	byPkg := make(map[string]map[string]struct{})

	entries, err := os.ReadDir(dir)
	if err != nil {
		return byPkg
	}

	fset := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		f, err := parser.ParseFile(fset, filepath.Join(dir, entry.Name()), nil, parser.SkipObjectResolution)
		if err != nil {
			continue
		}

		set, ok := byPkg[f.Name.Name]
		if !ok {
			set = make(map[string]struct{})
			byPkg[f.Name.Name] = set
		}
		collectConstNames(f, set)
	}

	return byPkg
}

// collectConstNames adds every package-level constant name declared by f to
// set. Only top-level declarations are package-level; a constant declared
// inside a function is resolved by the parser already.
func collectConstNames(f *ast.File, set map[string]struct{}) {
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}

		for _, spec := range gen.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			for _, name := range value.Names {
				set[name.Name] = struct{}{}
			}
		}
	}
}

// resolvePackageConsts attaches a constant object to each unresolved
// identifier whose name is a package-level constant, so the detector sees it
// as a constant rather than a bare variable. Identifiers the parser already
// resolved (locals, same-file declarations) are absent from f.Unresolved and
// are left untouched.
func resolvePackageConsts(f *ast.File, names map[string]struct{}) {
	for _, ident := range f.Unresolved {
		if _, isConst := names[ident.Name]; isConst {
			ident.Obj = ast.NewObj(ast.Con, ident.Name)
		}
	}
}

package assumpgo

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// parseExpr parses a single Go expression into an ast.Node.
func parseExpr(t *testing.T, src string) ast.Node {
	t.Helper()
	expr, err := parser.ParseExpr(src)
	if err != nil {
		t.Fatalf("parse %q: %v", src, err)
	}
	return expr
}

// parseStmt parses a single statement by wrapping it in a function body.
func parseStmt(t *testing.T, src string) ast.Stmt {
	t.Helper()
	wrapped := "package p\nfunc _() {\n" + src + "\n}\n"
	f, err := parser.ParseFile(token.NewFileSet(), "", wrapped, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse %q: %v", src, err)
	}
	body := f.Decls[0].(*ast.FuncDecl).Body
	return body.List[0]
}

func TestNewDetectorNotNil(t *testing.T) {
	if NewDetector() == nil {
		t.Fatal("NewDetector() returned nil")
	}
}

func TestScanDetectsNotEqual(t *testing.T) {
	d := NewDetector()
	for _, src := range []string{"x != nil", "nil != x", "x != 0", `x != "test"`} {
		if !d.Scan(parseExpr(t, src)) {
			t.Errorf("expected %q to be an assumption", src)
		}
	}
}

func TestScanDetectsBooleanNotVariable(t *testing.T) {
	d := NewDetector()
	if !d.Scan(parseExpr(t, "!x")) {
		t.Error("expected !x to be an assumption")
	}
	if d.Scan(parseExpr(t, "!true")) {
		t.Error("expected !true to be ignored (literal, not a variable)")
	}
	if d.Scan(parseExpr(t, "!helper(x)")) {
		t.Error("expected !helper(x) to be ignored (call, not a variable)")
	}
	// A unary expression that is not a boolean-not must never be flagged,
	// even when its operand is a variable.
	for _, src := range []string{"-x", "*p", "&x", "<-ch"} {
		if d.Scan(parseExpr(t, src)) {
			t.Errorf("expected %q (non-`!` unary) to be ignored", src)
		}
	}
}

func TestScanDetectsBareVariableConditions(t *testing.T) {
	d := NewDetector()
	if !d.Scan(parseStmt(t, "if x {}")) {
		t.Error("expected `if x` to be an assumption")
	}
	if !d.Scan(parseStmt(t, "for x {}")) {
		t.Error("expected `for x` to be an assumption")
	}
	if !d.Scan(parseStmt(t, "if doSomething(); running {}")) {
		t.Error("expected `if doSomething(); running` to be an assumption")
	}
	if !d.Scan(parseStmt(t, "for i := 0; running; i++ {}")) {
		t.Error("expected `for i := 0; running; i++` to be an assumption")
	}
	if !d.Scan(parseStmt(t, "if x := 1; running {}")) {
		t.Error("expected `if x := 1; running` to be an assumption")
	}
	if !d.Scan(parseStmt(t, "if running := isRunning(); running {}")) {
		t.Error("expected `if running := isRunning(); running` to be an assumption")
	}
	if !d.Scan(parseStmt(t, "if val, ok := m[k]; val {}")) {
		t.Error("expected non-ok variable in comma-ok to be an assumption")
	}
	if d.Scan(parseStmt(t, "if x == y {}")) {
		t.Error("expected `if x == y` not to be flagged via the bare-variable rule")
	}
}

func TestScanDetectsLogicalWithVariable(t *testing.T) {
	d := NewDetector()
	for _, src := range []string{`x && x == "test"`, `x == "test" && x`, "x || y != nil"} {
		if !d.Scan(parseExpr(t, src)) {
			t.Errorf("expected %q to be an assumption", src)
		}
	}
}

// TestScanDetectsLeftAssociativeVarComparisonMix covers issue #39: extra bare
// variables to the left of a mix must not hide it. `x && y && n == 1` parses
// as `(x && y) && (n == 1)`, so both outer operands are binary.
func TestScanDetectsLeftAssociativeVarComparisonMix(t *testing.T) {
	d := NewDetector()
	for _, src := range []string{
		"x && y && n == 1",
		"(x && y) && n == 1",
		"x && y && z && n == 1",
		"x || y || n == 1",
		"x && y || n == 1",
		"n == 1 && x && y",
		"x && (y && n == 1)",
	} {
		if !d.Scan(parseExpr(t, src)) {
			t.Errorf("expected %q to be an assumption", src)
		}
	}
}

// TestScanLogicalRequiresVariableAndComparison checks both halves of the
// bidirectional rule: a `&&`/`||` is only an assumption when exactly one side
// is a bare variable and the other is a (binary) comparison.
func TestScanLogicalRequiresVariableAndComparison(t *testing.T) {
	d := NewDetector()
	// Two bare variables: no comparison, not an assumption.
	for _, src := range []string{"x && y", "x || y"} {
		if d.Scan(parseExpr(t, src)) {
			t.Errorf("expected %q (two variables) not to be an assumption", src)
		}
	}
	// Chains of bare variables (issue #16): no comparison anywhere in the
	// chain, so not an assumption. `x && y && z` parses as `(x && y) && z`,
	// whose outer node has a bare variable on one side and a logical binary
	// expression on the other.
	for _, src := range []string{
		"x && y && z",
		"x || y || z",
		"x && y && z && w",
		"x || y || z || w",
		"x && y || z",
		"x && (y && z)",
	} {
		if d.Scan(parseExpr(t, src)) {
			t.Errorf("expected %q (chain of bare variables) not to be an assumption", src)
		}
	}
	// Two comparisons: no bare variable, not an assumption.
	for _, src := range []string{`x == 1 && y == 2`, `x != 1 || y != 2`, `x == 1 && y == 2 && z == 3`} {
		if d.Scan(parseExpr(t, src)) {
			t.Errorf("expected %q (two comparisons) not to be an assumption", src)
		}
	}
}

// TestScanIgnoresStrictEquality documents the deliberate difference from
// php-assumptions: Go's `==` is strict (the analog of PHP's `===`, which
// php-assumptions does not flag), so positive equality is treated as an
// assertion.
func TestScanIgnoresStrictEquality(t *testing.T) {
	d := NewDetector()
	for _, src := range []string{`x == "test"`, "x == nil", "x == y"} {
		if d.Scan(parseExpr(t, src)) {
			t.Errorf("expected %q not to be an assumption", src)
		}
	}
}

func TestScanIgnoresTypeAssertion(t *testing.T) {
	d := NewDetector()
	validAssertions := []string{
		"if _, ok := v.(*Dog); ok {}",
		"if val, ok := m[k]; ok {}",
		"if _, ok = v.(*Dog); ok {}",
		"for _, ok := m[k]; ok; {}",
		"if _, (ok) := v.(*Dog); (ok) {}",
	}
	for _, src := range validAssertions {
		if d.Scan(parseStmt(t, src)) {
			t.Errorf("expected %q not to be an assumption", src)
		}
	}
}

func TestScanIgnoresCommaOkChannelReceive(t *testing.T) {
	d := NewDetector()

	// Channel receive is the third comma-ok form: it binds `ok` the same way
	// type assertions and map indexes do (issue #33), so `ok` / `!ok` in the
	// condition is an assertion, not an assumption.
	valid := []string{
		"if v, ok := <-ch; ok {}",
		"if v, ok := <-ch; !ok {}",
		"if v, ok = <-ch; ok {}",
		"if v, ok = <-ch; !ok {}",
		"for v, ok := <-ch; ok; {}",
		"if v, ok := (<-ch); ok {}",
		"if v, (ok) := <-ch; (ok) {}",
	}
	for _, src := range valid {
		if d.Scan(parseStmt(t, src)) {
			t.Errorf("expected %q not to be an assumption", src)
		}
	}

	// Near-miss: a unary expression that is not a channel receive does not
	// bind `ok`, so the condition stays an assumption.
	if !d.Scan(parseStmt(t, "if v, ok := -x; ok {}")) {
		t.Error("expected `if v, ok := -x; ok` to be an assumption")
	}
	if !d.Scan(parseStmt(t, "if v, ok := <-ch; v {}")) {
		t.Error("expected non-ok variable in channel comma-ok to be an assumption")
	}
}

func TestInvertedCommaOkCond(t *testing.T) {
	d := NewDetector()

	// Positive cases: valid inverted comma-ok assertions.
	valid := []string{
		"if _, ok := v.(*Dog); !ok {}",
		"if val, ok := m[k]; !ok {}",
		"if v, ok := <-ch; !ok {}",
		"if v, ok := (<-ch); !ok {}",
		"if _, ok = v.(*Dog); !ok {}",
		"if _, ok := v.(*Dog); (!ok) {}",
		"if _, ok := (v.(*Dog)); !ok {}",
		"if _, ok := v.(*Dog); !(ok) {}",
		"if _, (ok) := v.(*Dog); !ok {}",
		"for _, ok := m[k]; !ok; {}",
	}
	for _, src := range valid {
		stmt := parseStmt(t, src)
		var cond ast.Expr
		switch s := stmt.(type) {
		case *ast.IfStmt:
			cond = d.invertedCommaOkCond(s.Init, s.Cond)
		case *ast.ForStmt:
			cond = d.invertedCommaOkCond(s.Init, s.Cond)
		}
		if cond == nil {
			t.Errorf("expected %q to be recognized as inverted comma-ok", src)
		}
	}

	// Negative cases: near-misses that must not be recognized as inverted comma-ok.
	invalid := []string{
		"if _, ok := v.(*Dog); ok {}",       // positive, not inverted
		"if _, ok := v.(*Dog); !other {}",   // different variable
		"if ok := check(); !ok {}",          // single variable init, not comma-ok
		"if val, ok := fn(); !ok {}",        // function call, not comma-ok expr
		"if !ok {}",                         // nil init
		"if _, _ := v.(*Dog); true {}",      // blank ok identifier
		"if a, b, c := m[k]; !c {}",         // 3 variables
		"if a := 1; !a {}",                  // 1 variable
		"if a, b = 1, 2; !b {}",             // 2 RHS expressions
		"if a, b[0] = v.(*Dog); !b {}",      // LHS[1] is index, not ident
		"if a, b := v.(*Dog); b == true {}", // binary expr condition, not unary NOT
		"if a, b := v.(*Dog); -b {}",        // unary op is not token.NOT
		"if a, b := v.(*Dog); !fn() {}",     // unary operand is not variable ident
	}
	for _, src := range invalid {
		stmt := parseStmt(t, src)
		var cond ast.Expr
		switch s := stmt.(type) {
		case *ast.IfStmt:
			cond = d.invertedCommaOkCond(s.Init, s.Cond)
		case *ast.ForStmt:
			cond = d.invertedCommaOkCond(s.Init, s.Cond)
		}
		if cond != nil {
			t.Errorf("expected %q not to be recognized as inverted comma-ok", src)
		}
	}

	// Direct nil checks on invertedCommaOkCond and commaOkVarName.
	if d.invertedCommaOkCond(nil, nil) != nil {
		t.Error("expected nil init/cond to return nil")
	}
	if commaOkVarName(nil) != "" {
		t.Error("expected nil init to return empty string")
	}
	// Init statement that is not an assignment.
	if commaOkVarName(parseStmt(t, "var x int")) != "" {
		t.Error("expected non-assignment init to return empty string")
	}
	// Condition without init.
	if d.invertedCommaOkCond(parseStmt(t, "x := 1"), nil) != nil {
		t.Error("expected nil cond to return nil")
	}
}

func TestIsBoolExpression(t *testing.T) {
	d := NewDetector()
	cases := map[string]bool{
		"if x == y {}":         true,
		"for x {}":             true,
		"for i := 0; ; i++ {}": false, // no condition
		"for {}":               false,
	}
	for src, want := range cases {
		got := d.IsBoolExpression(parseStmt(t, src))
		if got != want {
			t.Errorf("IsBoolExpression(%q) = %v, want %v", src, got, want)
		}
	}

	if !d.IsBoolExpression(parseExpr(t, "x && y")) {
		t.Error("expected && to be a boolean expression")
	}
	if !d.IsBoolExpression(parseExpr(t, "x || y")) {
		t.Error("expected || to be a boolean expression")
	}

	// Every node Scan can flag must also count as a boolean expression, or the
	// percentage denominator misses it (issue #34).
	if !d.IsBoolExpression(parseExpr(t, "x != y")) {
		t.Error("expected != to be a boolean expression (it is always an assumption)")
	}
	if !d.IsBoolExpression(parseExpr(t, "!x")) {
		t.Error("expected !x to be a boolean expression (it is always an assumption)")
	}

	// Near-misses: assertions and non-variable boolean-nots are neither
	// assumptions nor boolean expressions.
	for _, src := range []string{"x == y", "x == nil", "!helper(x)", "!true", "-x", "<-ch"} {
		if d.IsBoolExpression(parseExpr(t, src)) {
			t.Errorf("expected %q not to count as a boolean expression", src)
		}
	}
}

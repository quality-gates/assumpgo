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
	// Two comparisons: no bare variable, not an assumption.
	for _, src := range []string{`x == 1 && y == 2`, `x != 1 || y != 2`} {
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
	if d.Scan(parseStmt(t, "if _, ok := v.(*Dog); ok {}")) {
		t.Error("expected a type assertion guard not to be an assumption")
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
	if d.IsBoolExpression(parseExpr(t, "x != y")) {
		t.Error("expected != not to count as a boolean expression for the denominator")
	}
}

package assumpgo

import (
	"go/ast"
	"go/token"
)

// Detector decides whether a given AST node represents a weak "assumption"
// (the negative/blacklisting style of boolean check the "From assumptions to
// assertions" blog post argues against) and whether a node is a boolean
// expression at all (used to compute the assumption percentage).
//
// It is a direct port of the rules in rskuipers/php-assumptions, adapted to
// Go's single, strict set of comparison operators:
//
//   - PHP flags the loose `==`, the loose `!=` and the strict-negative `!==`,
//     but deliberately does NOT flag the strict-positive `===`.
//   - Go has only one equality operator. `==` is already strict (the analog of
//     PHP's `===`), so it is treated as an assertion and is NOT flagged.
//     `!=` is the negative, blacklisting comparison (the `$user !== null` from
//     the blog post) and IS flagged.
type Detector struct{}

// NewDetector returns a ready to use Detector.
func NewDetector() *Detector {
	return &Detector{}
}

// Scan reports whether node is a weak assumption.
func (d *Detector) Scan(node ast.Node) bool {
	switch n := node.(type) {
	case *ast.BinaryExpr:
		// `&&` / `||` where one operand is a bare variable and the other is a
		// (comparison or logical) expression, e.g. `x && x == "test"`.
		if n.Op == token.LAND || n.Op == token.LOR {
			return d.bidirectionalCheck(n)
		}
		// A negative comparison, e.g. `x != nil`. Mirrors PHP's NotEqual /
		// NotIdentical. The strict-positive `==` is treated as an assertion.
		if n.Op == token.NEQ {
			return true
		}
	}

	return d.isVariableExpression(node)
}

// IsBoolExpression reports whether node contributes to the boolean expression
// count (the denominator of the assumption percentage). Mirrors the PHP set of
// If / ElseIf / While / For / Ternary / && / || nodes. Go has no ternary and
// folds while/else-if into for/if.
func (d *Detector) IsBoolExpression(node ast.Node) bool {
	switch n := node.(type) {
	case *ast.IfStmt:
		return true
	case *ast.ForStmt:
		// A bare `for {}` has no condition and is not a boolean expression.
		return n.Cond != nil
	case *ast.BinaryExpr:
		return n.Op == token.LAND || n.Op == token.LOR
	}

	return false
}

// isVariableExpression covers the cases where a bare variable is (ab)used as a
// boolean: `!x`, `if x`, `for x`.
func (d *Detector) isVariableExpression(node ast.Node) bool {
	switch n := node.(type) {
	case *ast.UnaryExpr:
		if n.Op == token.NOT && isVarIdent(n.X) {
			return true
		}
	case *ast.IfStmt:
		// `if x { ... }` is a bare-variable assumption, but the comma-ok /
		// assignment idiom (`if _, ok := v.(*Dog); ok`) is the idiomatic Go
		// *assertion*, so we ignore bare conditions that bind their variable in
		// the Init statement.
		return n.Init == nil && isVarIdent(n.Cond)
	case *ast.ForStmt:
		return n.Init == nil && isVarIdent(n.Cond)
	}

	return false
}

// bidirectionalCheck reports whether one operand of a `&&`/`||` is a bare
// variable and the other is a binary expression, in either order. Mirrors the
// PHP Variable + BinaryOp bidirectional check.
func (d *Detector) bidirectionalCheck(n *ast.BinaryExpr) bool {
	left := unwrap(n.X)
	right := unwrap(n.Y)

	return (isVarIdent(left) && isBinary(right)) || (isVarIdent(right) && isBinary(left))
}

// isVarIdent reports whether expr is a bare identifier that refers to a
// variable, excluding the predeclared literals true/false/nil/iota (the Go
// analog of PHP distinguishing a Variable from a ConstFetch).
func isVarIdent(expr ast.Node) bool {
	ident, ok := unwrap(expr).(*ast.Ident)
	if !ok {
		return false
	}

	switch ident.Name {
	case "true", "false", "nil", "iota":
		return false
	}

	return true
}

func isBinary(expr ast.Node) bool {
	_, ok := expr.(*ast.BinaryExpr)
	return ok
}

// unwrap strips redundant parentheses so `(x)` is treated like `x`.
func unwrap(node ast.Node) ast.Node {
	for {
		paren, ok := node.(*ast.ParenExpr)
		if !ok {
			return node
		}
		node = paren.X
	}
}

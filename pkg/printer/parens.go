package printer

import "github.com/startracex/tsgo/pkg/ast"

// needsParens reports whether parentheses around n are required for the
// reprint to parse the same way. It is consulted only for parentheses that
// are already in the source, so a true result keeps them and a false result
// drops them. Unrecognized cases return true.
func needsParens(n *ast.Node) bool {
	if n == nil {
		return false
	}
	parent := syntacticParent(n)
	if parent == nil {
		return false
	}
	switch n.Kind {
	case ast.KindIdentifier, ast.KindThisKeyword, ast.KindSuperKeyword,
		ast.KindNullKeyword, ast.KindTrueKeyword, ast.KindFalseKeyword,
		ast.KindStringLiteral, ast.KindNoSubstitutionTemplateLiteral,
		ast.KindBigIntLiteral, ast.KindRegularExpressionLiteral,
		ast.KindArrayLiteralExpression, ast.KindMetaProperty,
		ast.KindImportKeyword:
		return false
	case ast.KindNumericLiteral:
		return isMemberObject(parent, n)
	case ast.KindObjectLiteralExpression:
		return objectNeedsParens(n, parent)
	case ast.KindFunctionExpression, ast.KindClassExpression:
		return exprNeedsParens(n, parent)
	case ast.KindPropertyAccessExpression, ast.KindElementAccessExpression,
		ast.KindCallExpression, ast.KindNonNullExpression:
		return leftHandNeedsParens(n, parent)
	case ast.KindBinaryExpression:
		return binaryChildNeedsParens(n, parent)
	case ast.KindParenthesizedExpression:
		return false
	default:
		return true
	}
}

func syntacticParent(n *ast.Node) *ast.Node {
	p := n.Parent
	for p != nil && p.Kind == ast.KindParenthesizedExpression {
		p = p.Parent
	}
	return p
}

func objectNeedsParens(n, parent *ast.Node) bool {
	switch parent.Kind {
	case ast.KindExpressionStatement, ast.KindArrowFunction, ast.KindExportAssignment:
		return true
	default:
		return leftHandNeedsParens(n, parent) || binaryChildNeedsParens(n, parent)
	}
}

func exprNeedsParens(n, parent *ast.Node) bool {
	switch parent.Kind {
	case ast.KindExpressionStatement, ast.KindCallExpression, ast.KindNewExpression,
		ast.KindPropertyAccessExpression, ast.KindElementAccessExpression,
		ast.KindTaggedTemplateExpression:
		return true
	default:
		return binaryChildNeedsParens(n, parent)
	}
}

func leftHandNeedsParens(n, parent *ast.Node) bool {
	switch parent.Kind {
	case ast.KindNewExpression:
		return isNewCallee(parent, n) && !canBeNewCallee(n)
	case ast.KindCallExpression:
		return isCallee(parent, n) && !canBeCallee(n)
	case ast.KindTaggedTemplateExpression:
		return isMemberObject(parent, n) && !canBeMemberObject(n)
	default:
		return false
	}
}

func binaryChildNeedsParens(child, parent *ast.Node) bool {
	if parent.Kind == ast.KindConditionalExpression {
		return nodePrec(child) <= precConditional
	}
	if parent.Kind != ast.KindBinaryExpression {
		return false
	}
	b := parent.AsBinaryExpression()
	op := b.OperatorToken.Kind
	if isAssign(op) {
		if sameTarget(b.Right, child) {
			return false
		}
		return nodePrec(child) < prec(op)
	}
	cp := nodePrec(child)
	opPrec := prec(op)
	if cp > opPrec {
		return false
	}
	if cp < opPrec {
		return true
	}
	left := sameTarget(b.Left, child)
	if op == ast.KindAsteriskAsteriskToken {
		return left
	}
	return !left
}

func sameTarget(slot, n *ast.Node) bool {
	if slot == nil || n == nil {
		return false
	}
	if slot == n {
		return true
	}
	if slot.Kind == ast.KindParenthesizedExpression {
		return slot.AsParenthesizedExpression().Expression == n
	}
	return false
}

func isCallee(call, n *ast.Node) bool {
	expr := call.AsCallExpression().Expression
	return expr == n || unwrapParen(expr) == n
}

func isNewCallee(nw, n *ast.Node) bool {
	expr := nw.AsNewExpression().Expression
	return expr == n || unwrapParen(expr) == n
}

func isMemberObject(member, n *ast.Node) bool {
	var obj *ast.Node
	switch member.Kind {
	case ast.KindPropertyAccessExpression:
		obj = member.AsPropertyAccessExpression().Expression
	case ast.KindElementAccessExpression:
		obj = member.AsElementAccessExpression().Expression
	case ast.KindTaggedTemplateExpression:
		obj = member.AsTaggedTemplateExpression().Tag
	default:
		return false
	}
	return obj == n || unwrapParen(obj) == n
}

func unwrapParen(n *ast.Node) *ast.Node {
	if n != nil && n.Kind == ast.KindParenthesizedExpression {
		return n.AsParenthesizedExpression().Expression
	}
	return n
}

func canBeCallee(n *ast.Node) bool {
	switch n.Kind {
	case ast.KindIdentifier, ast.KindPropertyAccessExpression, ast.KindElementAccessExpression,
		ast.KindCallExpression, ast.KindThisKeyword, ast.KindSuperKeyword, ast.KindImportKeyword,
		ast.KindNonNullExpression, ast.KindMetaProperty:
		return true
	case ast.KindNewExpression:
		return n.AsNewExpression().Arguments != nil
	default:
		return false
	}
}

func canBeNewCallee(n *ast.Node) bool {
	switch n.Kind {
	case ast.KindIdentifier, ast.KindPropertyAccessExpression, ast.KindElementAccessExpression,
		ast.KindThisKeyword, ast.KindSuperKeyword, ast.KindImportKeyword, ast.KindNonNullExpression,
		ast.KindCallExpression, ast.KindMetaProperty:
		return true
	default:
		return false
	}
}

func canBeMemberObject(n *ast.Node) bool {
	switch n.Kind {
	case ast.KindIdentifier, ast.KindPropertyAccessExpression, ast.KindElementAccessExpression,
		ast.KindCallExpression, ast.KindThisKeyword, ast.KindSuperKeyword, ast.KindImportKeyword,
		ast.KindNonNullExpression, ast.KindNewExpression, ast.KindMetaProperty:
		return true
	default:
		return false
	}
}

const precConditional = 3

func nodePrec(n *ast.Node) int {
	if n == nil {
		return 19
	}
	switch n.Kind {
	case ast.KindBinaryExpression:
		return prec(n.AsBinaryExpression().OperatorToken.Kind)
	case ast.KindConditionalExpression:
		return precConditional
	case ast.KindYieldExpression, ast.KindArrowFunction:
		return 2
	case ast.KindPrefixUnaryExpression, ast.KindAwaitExpression, ast.KindTypeOfExpression,
		ast.KindVoidExpression, ast.KindDeleteExpression:
		return 16
	case ast.KindPostfixUnaryExpression:
		return 17
	case ast.KindCallExpression, ast.KindPropertyAccessExpression, ast.KindElementAccessExpression,
		ast.KindNewExpression, ast.KindTaggedTemplateExpression, ast.KindNonNullExpression:
		return 18
	default:
		return 19
	}
}

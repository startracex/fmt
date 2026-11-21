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
	case ast.KindConditionalExpression, ast.KindConditionalType:
		// Nested branches do not need parens. A flat consequent still gets
		// them from the ternary printer, only while the chain fits one line.
		return !ternaryBranch(n, parent)
	case ast.KindParenthesizedExpression:
		return false
	default:
		return true
	}
}

func ternaryBranch(n, parent *ast.Node) bool {
	switch parent.Kind {
	case ast.KindConditionalExpression:
		c := parent.AsConditionalExpression()
		if sameTarget(c.Condition, n) {
			return false
		}
		return sameTarget(c.WhenTrue, n) || sameTarget(c.WhenFalse, n)
	case ast.KindConditionalType:
		c := parent.AsConditionalTypeNode()
		if c.CheckType == n || c.ExtendsType == n {
			return false
		}
		return c.TrueType == n || c.FalseType == n
	default:
		return false
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
	if child == nil || child.Kind != ast.KindBinaryExpression {
		return false
	}
	switch parent.Kind {
	case ast.KindPrefixUnaryExpression, ast.KindPostfixUnaryExpression,
		ast.KindAwaitExpression, ast.KindTypeOfExpression, ast.KindVoidExpression,
		ast.KindDeleteExpression, ast.KindTypeAssertionExpression,
		ast.KindNonNullExpression, ast.KindSpreadElement, ast.KindSpreadAssignment,
		ast.KindTaggedTemplateExpression, ast.KindAsExpression, ast.KindSatisfiesExpression:
		return true
	case ast.KindCallExpression:
		return isCallee(parent, child)
	case ast.KindNewExpression:
		return isNewCallee(parent, child)
	case ast.KindPropertyAccessExpression, ast.KindElementAccessExpression:
		return isMemberObject(parent, child)
	case ast.KindConditionalExpression:
		op := child.AsBinaryExpression().OperatorToken.Kind
		// Assignment and comma bind looser than ?:, so the branches need parens.
		if isAssign(op) || op == ast.KindCommaToken {
			return true
		}
		return op == ast.KindQuestionQuestionToken
	case ast.KindBinaryExpression:
		return binaryInBinary(child, parent)
	default:
		return false
	}
}

// binaryInBinary matches Prettier 3.6.2's decisions for a parenthesized
// binary nested in another binary. Same-operator + * && || & | chains drop
// parens on both sides. / % ** and shifts do not.
func binaryInBinary(child, parent *ast.Node) bool {
	cop := child.AsBinaryExpression().OperatorToken.Kind
	pop := parent.AsBinaryExpression().OperatorToken.Kind
	if isLogicalOp(cop) && isLogicalOp(pop) && cop != pop {
		return true
	}
	cp, pp := prec(cop), prec(pop)
	if pp > cp {
		return true
	}
	if pp < cp {
		if cop == ast.KindPercentToken && (pop == ast.KindPlusToken || pop == ast.KindMinusToken) {
			return true
		}
		if isBitwiseOp(pop) {
			return true
		}
		return false
	}
	right := sameTarget(parent.AsBinaryExpression().Right, child)
	if right {
		return !flattenRight(pop, cop)
	}
	return !flattenLeft(pop, cop)
}

func flattenLeft(parent, child ast.Kind) bool {
	if parent == child {
		switch parent {
		case ast.KindPercentToken, ast.KindAsteriskAsteriskToken,
			ast.KindLessThanLessThanToken, ast.KindGreaterThanGreaterThanToken,
			ast.KindGreaterThanGreaterThanGreaterThanToken:
			return false
		default:
			return true
		}
	}
	return isPlusMinus(parent) && isPlusMinus(child)
}

func flattenRight(parent, child ast.Kind) bool {
	if parent != child {
		return false
	}
	switch parent {
	case ast.KindPlusToken, ast.KindMinusToken, ast.KindAsteriskToken,
		ast.KindAmpersandAmpersandToken, ast.KindBarBarToken, ast.KindQuestionQuestionToken,
		ast.KindAmpersandToken, ast.KindBarToken, ast.KindCaretToken:
		return true
	default:
		return false
	}
}

func isLogicalOp(k ast.Kind) bool {
	return k == ast.KindAmpersandAmpersandToken || k == ast.KindBarBarToken || k == ast.KindQuestionQuestionToken
}

func isPlusMinus(k ast.Kind) bool {
	return k == ast.KindPlusToken || k == ast.KindMinusToken
}

func isBitwiseOp(k ast.Kind) bool {
	switch k {
	case ast.KindAmpersandToken, ast.KindBarToken, ast.KindCaretToken,
		ast.KindLessThanLessThanToken, ast.KindGreaterThanGreaterThanToken,
		ast.KindGreaterThanGreaterThanGreaterThanToken:
		return true
	default:
		return false
	}
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

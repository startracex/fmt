// Package printer turns a tsgo AST into a pretty-document.
//
// It does not call tsgo's language-service formatter. Comments are recovered
// from the gaps between tokens and placed back onto the reprinted tree.
package printer

import (
	"strings"
	"unsafe"

	"github.com/startracex/tsgo/pkg/ast"
	"github.com/startracex/tsgo/pkg/scanner"

	"github.com/startracex/fmt/doc"
	"github.com/startracex/fmt/pkg/opt"
)

// Printer builds a document for one source file.
type Printer struct {
	src      string
	opt      opt.Options
	comments []comment
	ci       int
	lastEnd  int
	// forceQuote makes every string-like property name quoted.
	forceQuote bool
}

// New returns a printer for src.
func New(src string, options opt.Options) *Printer {
	return &Printer{src: src, opt: options.Normalized()}
}

// File prints a parsed source file.
func (p *Printer) File(sf *ast.SourceFile) *doc.Doc {
	p.comments = collectComments(p.src, sf.AsNode())
	p.ci = 0
	var parts []*doc.Doc
	base := 0
	if strings.HasPrefix(p.src, "#!") {
		lineEnd := strings.IndexAny(p.src, "\r\n")
		if lineEnd < 0 {
			lineEnd = len(p.src)
		}
		parts = append(parts, doc.T(p.src[:lineEnd]), doc.HardLine())
		base = lineEnd
		if base < len(p.src) && (p.src[base] == '\n' || p.src[base] == '\r') {
			if p.src[base] == '\r' && base+1 < len(p.src) && p.src[base+1] == '\n' {
				base += 2
			} else {
				base++
			}
		}
		p.discardUntil(base)
	}
	var stmts []*ast.Node
	if sf.Statements != nil {
		stmts = sf.Statements.Nodes
	}
	parts = append(parts, p.statements(stmts))
	rest := p.takeUntil(len(p.src) + 1)
	if len(rest) > 0 {
		if len(stmts) > 0 {
			parts = append(parts, doc.HardLine())
		}
		parts = append(parts, p.commentLines(rest))
	}
	return doc.Concat(parts...)
}

// PrintFile formats src, which must already have been parsed as sf.
func PrintFile(sf *ast.SourceFile, options opt.Options) string {
	p := New(sf.Text(), options)
	unit := strings.Repeat(" ", p.opt.TabWidth)
	if p.opt.UseTabs {
		unit = "\t"
	}
	return doc.Print(p.File(sf), doc.PrintOptions{
		Width:      p.opt.PrintWidth,
		IndentUnit: unit,
		TabWidth:   p.opt.TabWidth,
	})
}

func (p *Printer) node(n *ast.Node) *doc.Doc {
	if n == nil || ast.NodeIsMissing(n) {
		return nil
	}
	start := p.skip(n)
	lead := p.leading(p.takeUntil(start), start)
	body := p.body(n)
	rest := p.inlineComments(p.takeUntil(n.End()))
	if n.End() > p.lastEnd {
		p.lastEnd = n.End()
	}
	return doc.Concat(lead, body, rest)
}

func (p *Printer) body(n *ast.Node) *doc.Doc {
	switch n.Kind {
	case ast.KindIdentifier:
		return doc.T(n.AsIdentifier().Text)
	case ast.KindPrivateIdentifier:
		return doc.T(privateText(n))
	case ast.KindStringLiteral:
		return doc.T(p.stringLiteral(n))
	case ast.KindNumericLiteral, ast.KindBigIntLiteral, ast.KindRegularExpressionLiteral, ast.KindNoSubstitutionTemplateLiteral:
		return doc.T(p.raw(n))
	case ast.KindThisKeyword, ast.KindSuperKeyword, ast.KindImportKeyword,
		ast.KindNullKeyword, ast.KindTrueKeyword, ast.KindFalseKeyword:
		return doc.T(scanner.TokenToString(n.Kind))
	case ast.KindQualifiedName:
		q := n.AsQualifiedName()
		return doc.Concat(p.node(q.Left), doc.T("."), p.node(q.Right))
	case ast.KindComputedPropertyName:
		return doc.Concat(doc.T("["), p.node(n.AsComputedPropertyName().Expression), doc.T("]"))
	case ast.KindDecorator:
		return doc.Concat(doc.T("@"), p.node(n.AsDecorator().Expression))
	case ast.KindTemplateHead, ast.KindTemplateMiddle, ast.KindTemplateTail:
		return doc.T(p.raw(n))
	case ast.KindTemplateExpression:
		return p.template(n)
	case ast.KindTemplateSpan:
		s := n.AsTemplateSpan()
		return doc.Concat(p.node(s.Expression), p.node(s.Literal))
	default:
		if isTokenKind(n.Kind) {
			if s := scanner.TokenToString(n.Kind); s != "" {
				p.discardUntil(n.End())
				return doc.T(s)
			}
		}
		return p.dispatch(n)
	}
}

func (p *Printer) dispatch(n *ast.Node) *doc.Doc {
	switch n.Kind {
	case ast.KindBlock:
		return p.block(n)
	case ast.KindEmptyStatement:
		return nil
	case ast.KindVariableStatement:
		return p.variableStatement(n)
	case ast.KindVariableDeclarationList:
		return p.variableList(n)
	case ast.KindVariableDeclaration:
		return p.variable(n)
	case ast.KindExpressionStatement:
		return p.node(n.AsExpressionStatement().Expression)
	case ast.KindIfStatement:
		return p.ifStmt(n)
	case ast.KindDoStatement:
		return p.doStmt(n)
	case ast.KindWhileStatement:
		return p.whileStmt(n)
	case ast.KindForStatement:
		return p.forStmt(n)
	case ast.KindForInStatement, ast.KindForOfStatement:
		return p.forInOf(n)
	case ast.KindContinueStatement, ast.KindBreakStatement:
		return p.breakCont(n)
	case ast.KindReturnStatement:
		return p.returnStmt(n)
	case ast.KindWithStatement:
		w := n.AsWithStatement()
		return p.withBody("with", w.Expression, w.Statement)
	case ast.KindSwitchStatement:
		return p.switchStmt(n)
	case ast.KindCaseBlock:
		return p.caseBlock(n)
	case ast.KindCaseClause, ast.KindDefaultClause:
		return p.caseClause(n)
	case ast.KindThrowStatement:
		return doc.Concat(doc.T("throw "), p.node(n.AsThrowStatement().Expression))
	case ast.KindTryStatement:
		return p.tryStmt(n)
	case ast.KindCatchClause:
		return p.catchClause(n)
	case ast.KindDebuggerStatement:
		return doc.T("debugger")
	case ast.KindLabeledStatement:
		l := n.AsLabeledStatement()
		return doc.Concat(p.node(l.Label), doc.T(": "), p.embedded(l.Statement))
	case ast.KindFunctionDeclaration, ast.KindFunctionExpression:
		return p.function(n)
	case ast.KindClassDeclaration, ast.KindClassExpression:
		return p.class(n)
	case ast.KindInterfaceDeclaration:
		return p.iface(n)
	case ast.KindTypeAliasDeclaration, ast.KindJSTypeAliasDeclaration:
		return p.typeAlias(n)
	case ast.KindEnumDeclaration:
		return p.enum(n)
	case ast.KindEnumMember:
		return p.enumMember(n)
	case ast.KindModuleDeclaration:
		return p.module(n)
	case ast.KindModuleBlock:
		return p.moduleBlock(n)
	case ast.KindImportDeclaration, ast.KindJSImportDeclaration:
		return p.importDecl(n)
	case ast.KindImportEqualsDeclaration:
		return p.importEquals(n)
	case ast.KindExportAssignment:
		return p.exportAssign(n)
	case ast.KindExportDeclaration:
		return p.exportDecl(n)
	case ast.KindNamespaceExportDeclaration:
		d := n.AsNamespaceExportDeclaration()
		return doc.Concat(p.modifiers(n), doc.T("export as namespace "), p.node(d.Name()))
	case ast.KindImportClause:
		return p.importClause(n)
	case ast.KindNamedImports, ast.KindNamedExports:
		return p.namedList(n)
	case ast.KindImportSpecifier:
		return p.importSpec(n)
	case ast.KindExportSpecifier:
		return p.exportSpec(n)
	case ast.KindNamespaceImport:
		return doc.Concat(doc.T("* as "), p.node(n.AsNamespaceImport().Name()))
	case ast.KindNamespaceExport:
		return doc.Concat(doc.T("* as "), p.node(n.AsNamespaceExport().Name()))
	case ast.KindExternalModuleReference:
		return doc.Concat(doc.T("require("), p.node(n.AsExternalModuleReference().Expression), doc.T(")"))
	case ast.KindImportAttributes:
		return p.importAttrs(n)
	case ast.KindImportAttribute:
		a := n.AsImportAttribute()
		return doc.Concat(p.propName(a.Name()), doc.T(": "), p.node(a.Value))
	case ast.KindMissingDeclaration:
		return p.modifiers(n)

	case ast.KindBinaryExpression:
		return p.binary(n)
	case ast.KindPrefixUnaryExpression:
		return p.prefix(n)
	case ast.KindPostfixUnaryExpression:
		return p.postfix(n)
	case ast.KindConditionalExpression:
		return p.conditional(n, false)
	case ast.KindYieldExpression:
		return p.yield(n)
	case ast.KindArrowFunction:
		return p.arrow(n)
	case ast.KindParenthesizedExpression:
		return doc.Concat(doc.T("("), p.node(n.AsParenthesizedExpression().Expression), doc.T(")"))
	case ast.KindArrayLiteralExpression:
		return p.array(n)
	case ast.KindObjectLiteralExpression:
		return p.object(n)
	case ast.KindPropertyAccessExpression, ast.KindElementAccessExpression, ast.KindCallExpression, ast.KindTaggedTemplateExpression:
		return p.chain(n)
	case ast.KindNewExpression:
		return p.newExpr(n)
	case ast.KindAsExpression:
		a := n.AsAsExpression()
		return doc.Concat(p.node(a.Expression), doc.T(" as "), p.node(a.Type))
	case ast.KindSatisfiesExpression:
		a := n.AsSatisfiesExpression()
		return doc.Concat(p.node(a.Expression), doc.T(" satisfies "), p.node(a.Type))
	case ast.KindTypeAssertionExpression:
		a := n.AsTypeAssertion()
		return doc.Concat(doc.T("<"), p.node(a.Type), doc.T(">"), p.node(a.Expression))
	case ast.KindNonNullExpression:
		return doc.Concat(p.node(n.AsNonNullExpression().Expression), doc.T("!"))
	case ast.KindMetaProperty:
		m := n.AsMetaProperty()
		return doc.Concat(doc.T(scanner.TokenToString(m.KeywordToken)), doc.T("."), p.node(m.Name()))
	case ast.KindSpreadElement, ast.KindSpreadAssignment:
		return doc.Concat(doc.T("..."), p.node(p.spreadExpr(n)))
	case ast.KindDeleteExpression, ast.KindTypeOfExpression, ast.KindVoidExpression, ast.KindAwaitExpression:
		return p.wordUnary(n)
	case ast.KindPropertyAssignment:
		return p.propAssign(n)
	case ast.KindShorthandPropertyAssignment:
		return p.shorthand(n)
	case ast.KindOmittedExpression:
		return nil
	case ast.KindPartiallyEmittedExpression:
		return p.node(n.AsPartiallyEmittedExpression().Expression)

	case ast.KindParameter:
		return p.parameter(n)
	case ast.KindBindingElement:
		return p.bindingElem(n)
	case ast.KindObjectBindingPattern:
		return p.bindingPattern(n, "{", "}", opt.TrailObject)
	case ast.KindArrayBindingPattern:
		return p.bindingPattern(n, "[", "]", opt.TrailArray)
	case ast.KindTypeParameter:
		return p.typeParam(n)

	case ast.KindMethodDeclaration, ast.KindMethodSignature, ast.KindConstructor,
		ast.KindGetAccessor, ast.KindSetAccessor, ast.KindClassStaticBlockDeclaration,
		ast.KindPropertyDeclaration, ast.KindPropertySignature, ast.KindSemicolonClassElement,
		ast.KindIndexSignature, ast.KindCallSignature, ast.KindConstructSignature:
		return p.member(n)
	case ast.KindHeritageClause:
		return p.heritage(n)
	case ast.KindExpressionWithTypeArguments:
		e := n.AsExpressionWithTypeArguments()
		return doc.Concat(p.node(e.Expression), p.typeArgs(e.TypeArguments))

	case ast.KindJsxElement, ast.KindJsxSelfClosingElement, ast.KindJsxFragment,
		ast.KindJsxOpeningElement, ast.KindJsxClosingElement, ast.KindJsxAttribute,
		ast.KindJsxSpreadAttribute, ast.KindJsxExpression, ast.KindJsxText,
		ast.KindJsxNamespacedName, ast.KindJsxAttributes:
		return p.jsx(n)

	default:
		if n.Kind >= ast.KindFirstTypeNode && n.Kind <= ast.KindLastTypeNode {
			return p.typeNode(n)
		}
		return p.rawFallback(n)
	}
}

func (p *Printer) rawFallback(n *ast.Node) *doc.Doc {
	p.discardUntil(n.End())
	return doc.T(p.raw(n))
}

func (p *Printer) skip(n *ast.Node) int {
	if n == nil {
		return 0
	}
	pos := n.Pos()
	if pos < 0 || pos > len(p.src) {
		return 0
	}
	return scanner.SkipTrivia(p.src, pos)
}

func (p *Printer) raw(n *ast.Node) string {
	if n == nil {
		return ""
	}
	s := p.skip(n)
	e := n.End()
	if e > len(p.src) {
		e = len(p.src)
	}
	if s < 0 || s > e {
		return ""
	}
	return p.src[s:e]
}

func privateText(n *ast.Node) string {
	t := n.AsPrivateIdentifier().Text
	if strings.HasPrefix(t, "#") {
		return t
	}
	return "#" + t
}

func (p *Printer) stringLiteral(n *ast.Node) string {
	lit := n.AsStringLiteral()
	return quote(lit.Text, useSingle(lit.Text, p.opt.SingleQuote))
}

func (p *Printer) statements(nodes []*ast.Node) *doc.Doc {
	var parts []*doc.Doc
	var prev *ast.Node
	for i, n := range nodes {
		if n == nil || ast.NodeIsMissing(n) || n.Kind == ast.KindEndOfFile {
			continue
		}
		if prev != nil {
			if t := p.takeSameLine(prev.End(), p.skip(n)); t != nil {
				parts = append(parts, t)
			}
			parts = append(parts, doc.HardLine())
			if blankBetween(p.src, prev.End(), p.skip(n)) {
				parts = append(parts, doc.HardLine())
			}
		}
		parts = append(parts, p.finish(n, nextStmt(nodes, i+1)))
		prev = n
	}
	if prev != nil {
		if t := p.takeSameLine(prev.End(), len(p.src)+1); t != nil {
			parts = append(parts, t)
		}
	}
	return doc.Concat(parts...)
}

func nextStmt(nodes []*ast.Node, from int) *ast.Node {
	for _, n := range nodes[from:] {
		if n != nil && !ast.NodeIsMissing(n) && n.Kind != ast.KindEndOfFile {
			return n
		}
	}
	return nil
}

func (p *Printer) finish(n, next *ast.Node) *doc.Doc {
	d := p.node(n)
	if p.wantSemi(n, next) {
		return doc.Concat(d, doc.T(";"))
	}
	return d
}

func (p *Printer) wantSemi(n, next *ast.Node) bool {
	if !semiKind(n) {
		return false
	}
	if p.opt.Semi {
		return true
	}
	return p.startsDangerous(next)
}

// startsDangerous reports whether a following statement would be parsed as a
// continuation when semicolons are omitted.
func (p *Printer) startsDangerous(n *ast.Node) bool {
	if n == nil {
		return false
	}
	s := p.skip(n)
	if s >= len(p.src) {
		return false
	}
	switch p.src[s] {
	case '(', '[', '`', '+', '-', '/':
		return true
	default:
		return false
	}
}

func semiKind(n *ast.Node) bool {
	switch n.Kind {
	case ast.KindVariableStatement, ast.KindExpressionStatement, ast.KindReturnStatement,
		ast.KindThrowStatement, ast.KindBreakStatement, ast.KindContinueStatement,
		ast.KindDebuggerStatement, ast.KindImportDeclaration, ast.KindJSImportDeclaration,
		ast.KindExportDeclaration, ast.KindExportAssignment, ast.KindImportEqualsDeclaration,
		ast.KindTypeAliasDeclaration, ast.KindJSTypeAliasDeclaration,
		ast.KindNamespaceExportDeclaration, ast.KindEmptyStatement,
		ast.KindPropertyDeclaration, ast.KindPropertySignature, ast.KindMethodSignature,
		ast.KindCallSignature, ast.KindConstructSignature, ast.KindIndexSignature:
		return true
	case ast.KindMethodDeclaration, ast.KindConstructor, ast.KindGetAccessor, ast.KindSetAccessor:
		return n.Body() == nil
	case ast.KindModuleDeclaration:
		return n.Body() == nil
	default:
		return false
	}
}

func (p *Printer) modifiers(n *ast.Node) *doc.Doc {
	m := n.Modifiers()
	if m == nil || len(m.Nodes) == 0 {
		return nil
	}
	var parts []*doc.Doc
	for _, mod := range m.Nodes {
		if mod.Kind == ast.KindDecorator {
			if len(parts) > 0 {
				parts = append(parts, doc.HardLine())
			}
			parts = append(parts, p.node(mod), doc.HardLine())
			continue
		}
		if len(parts) > 0 {
			parts = append(parts, doc.T(" "))
		}
		parts = append(parts, p.node(mod))
	}
	if len(parts) == 0 {
		return nil
	}
	parts = append(parts, doc.T(" "))
	return doc.Concat(parts...)
}

func (p *Printer) propName(n *ast.Node) *doc.Doc {
	if n == nil {
		return nil
	}
	switch n.Kind {
	case ast.KindStringLiteral:
		text := n.AsStringLiteral().Text
		if p.quoteProp(text, true) {
			return doc.T(p.stringLiteral(n))
		}
		return doc.T(text)
	case ast.KindNoSubstitutionTemplateLiteral:
		return doc.T(p.raw(n))
	case ast.KindNumericLiteral, ast.KindBigIntLiteral:
		return doc.T(p.raw(n))
	case ast.KindComputedPropertyName:
		return p.node(n)
	case ast.KindPrivateIdentifier:
		return doc.T(privateText(n))
	default:
		return p.node(n)
	}
}

func (p *Printer) quoteProp(text string, wasString bool) bool {
	switch p.opt.QuoteProps {
	case "preserve":
		return wasString
	case "consistent":
		if p.forceQuote {
			return true
		}
		return !scanner.IsValidIdentifier(text)
	default:
		return !scanner.IsValidIdentifier(text)
	}
}

func listNodes(list *ast.NodeList) []*ast.Node {
	if list == nil {
		return nil
	}
	return list.Nodes
}

func isTokenKind(k ast.Kind) bool {
	return k >= ast.KindFirstToken && k <= ast.KindLastToken
}

func importData(n *ast.Node) *ast.ImportDeclaration {
	return (*ast.ImportDeclaration)(unsafe.Pointer(n))
}

func typeAliasData(n *ast.Node) *ast.TypeAliasDeclaration {
	return (*ast.TypeAliasDeclaration)(unsafe.Pointer(n))
}

package printer

import (
	"github.com/startracex/tsgo/pkg/ast"
	"github.com/startracex/tsgo/pkg/scanner"

	"github.com/startracex/fmt/doc"
	"github.com/startracex/fmt/pkg/opt"
)

func (p *Printer) block(n *ast.Node) *doc.Doc {
	b := n.AsBlock()
	stmts := listNodes(b.Statements)
	if len(filterStmts(stmts)) == 0 {
		dangle := p.commentLines(p.takeUntil(n.End()))
		if dangle != nil {
			return doc.Group(doc.Concat(doc.T("{"), doc.Indent(doc.Concat(doc.HardLine(), dangle)), doc.HardLine(), doc.T("}")))
		}
		if emptyBlockBreaks(n) {
			return doc.Concat(doc.T("{"), doc.HardLine(), doc.T("}"))
		}
		return doc.T("{}")
	}
	return p.braceBlock(stmts, n.End())
}

// emptyBlockBreaks matches Prettier: an empty block is `{}` for functions,
// classic for, while, and do, and `{` newline `}` for if, for-of, try, and
// other statement blocks.
func emptyBlockBreaks(n *ast.Node) bool {
	parent := n.Parent
	if parent == nil {
		return true
	}
	switch parent.Kind {
	case ast.KindArrowFunction, ast.KindFunctionExpression, ast.KindFunctionDeclaration,
		ast.KindMethodDeclaration, ast.KindConstructor, ast.KindGetAccessor, ast.KindSetAccessor,
		ast.KindForStatement, ast.KindWhileStatement, ast.KindDoStatement,
		ast.KindModuleDeclaration, ast.KindClassStaticBlockDeclaration:
		return false
	case ast.KindCatchClause:
		tryNode := parent.Parent
		if tryNode != nil && tryNode.Kind == ast.KindTryStatement && tryNode.AsTryStatement().FinallyBlock == nil {
			return false
		}
		return true
	default:
		return true
	}
}

func (p *Printer) moduleBlock(n *ast.Node) *doc.Doc {
	return p.braceBlock(listNodes(n.AsModuleBlock().Statements), n.End())
}

func (p *Printer) braceBlock(stmts []*ast.Node, end int) *doc.Doc {
	if len(filterStmts(stmts)) == 0 {
		dangle := p.commentLines(p.takeUntil(end))
		if dangle == nil {
			return doc.T("{}")
		}
		return doc.Group(doc.Concat(doc.T("{"), doc.Indent(doc.Concat(doc.HardLine(), dangle)), doc.HardLine(), doc.T("}")))
	}
	inner := p.statements(stmts)
	if dangle := p.commentLines(p.takeUntil(end)); dangle != nil {
		inner = doc.Concat(inner, doc.HardLine(), dangle)
	}
	// The group absorbs hard breaks so a parent (if, arrow, assignment)
	// can stay flat while the block itself still expands.
	return doc.Group(doc.Concat(doc.T("{"), doc.Indent(doc.Concat(doc.HardLine(), inner)), doc.HardLine(), doc.T("}")))
}

func filterStmts(nodes []*ast.Node) []*ast.Node {
	var out []*ast.Node
	for _, n := range nodes {
		if n != nil && !ast.NodeIsMissing(n) && n.Kind != ast.KindEndOfFile {
			out = append(out, n)
		}
	}
	return out
}

func (p *Printer) embedded(n *ast.Node) *doc.Doc {
	if n == nil {
		return nil
	}
	if n.Kind == ast.KindBlock {
		return doc.Concat(doc.T(" "), p.node(n))
	}
	return doc.Concat(doc.Indent(doc.Concat(doc.Line(), p.finish(n, nil))))
}

func (p *Printer) variableStatement(n *ast.Node) *doc.Doc {
	vs := n.AsVariableStatement()
	return doc.Concat(p.modifiers(n), p.variableList(vs.DeclarationList))
}

func declKeyword(flags ast.NodeFlags) string {
	switch flags & ast.NodeFlagsBlockScoped {
	case ast.NodeFlagsAwaitUsing:
		return "await using"
	case ast.NodeFlagsUsing:
		return "using"
	case ast.NodeFlagsConst:
		return "const"
	case ast.NodeFlagsLet:
		return "let"
	default:
		return "var"
	}
}

func (p *Printer) variableList(n *ast.Node) *doc.Doc {
	if n == nil {
		return nil
	}
	list := n.AsVariableDeclarationList()
	decls := listNodes(list.Declarations)
	kw := declKeyword(n.Flags)
	if len(decls) == 0 {
		return doc.T(kw)
	}
	first := p.node(decls[0])
	if len(decls) == 1 {
		return doc.Concat(doc.T(kw), doc.T(" "), first)
	}
	var rest []*doc.Doc
	for _, d := range decls[1:] {
		rest = append(rest, doc.T(","), doc.Line(), p.node(d))
	}
	return doc.Group(doc.Concat(doc.T(kw), doc.T(" "), first, doc.Indent(doc.Concat(rest...))))
}

func (p *Printer) variable(n *ast.Node) *doc.Doc {
	v := n.AsVariableDeclaration()
	return p.nameInit(p.binding(v.Name()), v.ExclamationToken != nil, v.Type, v.Initializer)
}

func (p *Printer) nameInit(name *doc.Doc, definite bool, typ, init *ast.Node) *doc.Doc {
	d := name
	if definite {
		d = doc.Concat(d, doc.T("!"))
	}
	if typ != nil {
		d = doc.Concat(d, doc.T(": "), p.node(typ))
	}
	if init != nil {
		if jsx, ok := jsxInner(init); ok {
			d = doc.Concat(d, p.jsxBreak(" = ", " = (", jsx))
		} else {
			d = p.withValue(d, " = ", " =", init)
		}
	}
	return d
}

func (p *Printer) binding(n *ast.Node) *doc.Doc {
	if n == nil {
		return nil
	}
	switch n.Kind {
	case ast.KindObjectBindingPattern:
		return p.bindingPattern(n, "{", "}", opt.TrailObject)
	case ast.KindArrayBindingPattern:
		return p.bindingPattern(n, "[", "]", opt.TrailArray)
	default:
		return p.node(n)
	}
}

func (p *Printer) bindingPattern(n *ast.Node, open, close string, trail opt.TrailKind) *doc.Doc {
	els := listNodes(n.AsBindingPattern().Elements)
	return p.commaGroup(open, close, els, n.End(), trail, p.opt.BracketSpacing && open == "{")
}

func (p *Printer) bindingElem(n *ast.Node) *doc.Doc {
	b := n.AsBindingElement()
	var d *doc.Doc
	if b.DotDotDotToken != nil {
		d = doc.T("...")
	}
	if b.PropertyName != nil {
		d = doc.Concat(d, p.propName(b.PropertyName), doc.T(": "), p.binding(b.Name()))
	} else {
		d = doc.Concat(d, p.binding(b.Name()))
	}
	if b.Initializer != nil {
		d = p.withValue(d, " = ", " =", b.Initializer)
	}
	return d
}

func (p *Printer) ifStmt(n *ast.Node) *doc.Doc {
	s := n.AsIfStatement()
	body := doc.Concat(doc.T("if ("), doc.Indent(doc.Concat(doc.SoftLine(), p.node(s.Expression))), doc.SoftLine(), doc.T(")"), p.embedded(s.ThenStatement))
	// A chain of braceless ifs breaks even when it would fit on one line.
	var d *doc.Doc
	if s.ThenStatement != nil && s.ThenStatement.Kind == ast.KindIfStatement {
		d = doc.BreakGroup(body)
	} else {
		d = doc.Group(body)
	}
	if s.ElseStatement == nil {
		return d
	}
	el := s.ElseStatement
	if el.Kind == ast.KindIfStatement {
		return doc.Concat(d, doc.T(" else "), p.node(el))
	}
	return doc.Concat(d, doc.T(" else"), p.embedded(el))
}

func (p *Printer) doStmt(n *ast.Node) *doc.Doc {
	s := n.AsDoStatement()
	body := s.Statement
	var b *doc.Doc
	if body != nil && body.Kind == ast.KindBlock {
		b = doc.Concat(doc.T("do "), p.node(body))
	} else {
		b = doc.Concat(doc.T("do"), p.embedded(body))
	}
	semi := ""
	if p.opt.Semi {
		semi = ";"
	}
	return doc.Concat(b, doc.T(" while ("), p.node(s.Expression), doc.T(")"+semi))
}

func (p *Printer) whileStmt(n *ast.Node) *doc.Doc {
	s := n.AsWhileStatement()
	return p.withBody("while", s.Expression, s.Statement)
}

func (p *Printer) withBody(kw string, expr, stmt *ast.Node) *doc.Doc {
	return doc.Group(doc.Concat(
		doc.T(kw+" ("),
		doc.Indent(doc.Concat(doc.SoftLine(), p.node(expr))),
		doc.SoftLine(),
		doc.T(")"),
		p.embedded(stmt),
	))
}

func (p *Printer) forStmt(n *ast.Node) *doc.Doc {
	s := n.AsForStatement()
	init := p.forInit(s.Initializer)
	cond := p.node(s.Condition)
	step := p.node(s.Incrementor)
	head := doc.Group(doc.Concat(
		doc.T("for ("),
		doc.Indent(doc.Concat(doc.SoftLine(), init, doc.T(";"), doc.Line(), cond, doc.T(";"), doc.Line(), step)),
		doc.SoftLine(),
		doc.T(")"),
	))
	return doc.Concat(head, p.embedded(s.Statement))
}

func (p *Printer) forInit(n *ast.Node) *doc.Doc {
	if n == nil {
		return nil
	}
	if n.Kind == ast.KindVariableDeclarationList {
		return p.variableList(n)
	}
	return p.node(n)
}

func (p *Printer) forInOf(n *ast.Node) *doc.Doc {
	s := n.AsForInOrOfStatement()
	kw := "for"
	if s.AwaitModifier != nil {
		kw = "for await"
	}
	op := " in "
	if n.Kind == ast.KindForOfStatement {
		op = " of "
	}
	head := doc.Group(doc.Concat(
		doc.T(kw+" ("),
		doc.Indent(doc.Concat(doc.SoftLine(), p.forInit(s.Initializer), doc.T(op), p.node(s.Expression))),
		doc.SoftLine(),
		doc.T(")"),
	))
	return doc.Concat(head, p.embedded(s.Statement))
}

func (p *Printer) breakCont(n *ast.Node) *doc.Doc {
	kw := "break"
	var label *ast.Node
	if n.Kind == ast.KindContinueStatement {
		kw = "continue"
		label = n.AsContinueStatement().Label
	} else {
		label = n.AsBreakStatement().Label
	}
	if label == nil {
		return doc.T(kw)
	}
	return doc.Concat(doc.T(kw), doc.T(" "), p.node(label))
}

func (p *Printer) returnStmt(n *ast.Node) *doc.Doc {
	e := n.AsReturnStatement().Expression
	if e == nil {
		return doc.T("return")
	}
	if jsx, ok := jsxInner(e); ok {
		return p.jsxBreak("return ", "return (", jsx)
	}
	return p.withValue(doc.T("return"), " ", "", e)
}

// jsxBreak prints a JSX expression. The flat form has no parentheses. The
// broken form wraps it so a newline after return or `=` is not parsed as ASI.
func (p *Printer) jsxBreak(flat, broken string, n *ast.Node) *doc.Doc {
	return doc.Group(doc.Concat(
		doc.IfBreak(doc.T(broken), doc.T(flat)),
		doc.Indent(doc.Concat(doc.SoftLine(), p.node(n))),
		doc.SoftLine(),
		doc.IfBreak(doc.T(")"), nil),
	))
}

func isJSX(n *ast.Node) bool {
	_, ok := jsxInner(n)
	return ok
}

// jsxInner unwraps parentheses around JSX so a formatted file parses back to
// the same shape.
func jsxInner(n *ast.Node) (*ast.Node, bool) {
	for n != nil && n.Kind == ast.KindParenthesizedExpression {
		n = n.AsParenthesizedExpression().Expression
	}
	if n == nil {
		return nil, false
	}
	switch n.Kind {
	case ast.KindJsxElement, ast.KindJsxSelfClosingElement, ast.KindJsxFragment:
		return n, true
	default:
		return nil, false
	}
}

func (p *Printer) switchStmt(n *ast.Node) *doc.Doc {
	s := n.AsSwitchStatement()
	return doc.Concat(
		doc.Group(doc.Concat(doc.T("switch ("), doc.Indent(doc.Concat(doc.SoftLine(), p.node(s.Expression))), doc.SoftLine(), doc.T(") "))),
		p.node(s.CaseBlock),
	)
}

func (p *Printer) caseBlock(n *ast.Node) *doc.Doc {
	clauses := listNodes(n.AsCaseBlock().Clauses)
	if len(clauses) == 0 {
		p.discardUntil(n.End())
		return doc.T("{}")
	}
	var parts []*doc.Doc
	for i, c := range clauses {
		if i > 0 {
			parts = append(parts, doc.HardLine())
		}
		parts = append(parts, p.node(c))
	}
	if dangle := p.commentLines(p.takeUntil(n.End())); dangle != nil {
		parts = append(parts, doc.HardLine(), dangle)
	}
	return doc.Group(doc.Concat(doc.T("{"), doc.Indent(doc.Concat(doc.HardLine(), doc.Concat(parts...))), doc.HardLine(), doc.T("}")))
}

func (p *Printer) caseClause(n *ast.Node) *doc.Doc {
	var head *doc.Doc
	var stmts *ast.NodeList
	if n.Kind == ast.KindCaseClause {
		c := n.AsCaseClause()
		head = doc.Concat(doc.T("case "), p.node(c.Expression), doc.T(":"))
		stmts = c.Statements
	} else {
		head = doc.T("default:")
		stmts = n.AsDefaultClause().Statements
	}
	body := p.statements(listNodes(stmts))
	if body == nil {
		return head
	}
	return doc.Concat(head, doc.Indent(doc.Concat(doc.HardLine(), body)))
}

func (p *Printer) tryStmt(n *ast.Node) *doc.Doc {
	s := n.AsTryStatement()
	d := doc.Concat(doc.T("try "), p.node(s.TryBlock))
	if s.CatchClause != nil {
		d = doc.Concat(d, doc.T(" "), p.node(s.CatchClause))
	}
	if s.FinallyBlock != nil {
		d = doc.Concat(d, doc.T(" finally "), p.node(s.FinallyBlock))
	}
	return d
}

func (p *Printer) catchClause(n *ast.Node) *doc.Doc {
	c := n.AsCatchClause()
	if c.VariableDeclaration == nil {
		return doc.Concat(doc.T("catch "), p.node(c.Block))
	}
	return doc.Concat(doc.T("catch ("), p.node(c.VariableDeclaration), doc.T(") "), p.node(c.Block))
}

func (p *Printer) function(n *ast.Node) *doc.Doc {
	var mods *doc.Doc
	var star *ast.Node
	var name *ast.Node
	var tparams, params *ast.NodeList
	var ret, body *ast.Node
	if n.Kind == ast.KindFunctionDeclaration {
		f := n.AsFunctionDeclaration()
		mods = p.modifiers(n)
		star = f.AsteriskToken
		name = f.Name()
		tparams = f.TypeParameters
		params = f.Parameters
		ret = f.Type
		body = f.Body
	} else {
		f := n.AsFunctionExpression()
		mods = p.modifiers(n)
		star = f.AsteriskToken
		name = f.Name()
		tparams = f.TypeParameters
		params = f.Parameters
		ret = f.Type
		body = f.Body
	}
	kw := "function"
	if star != nil {
		kw = "function*"
	}
	d := doc.Concat(mods, doc.T(kw))
	if name != nil {
		d = doc.Concat(d, doc.T(" "), p.node(name))
	} else {
		d = doc.Concat(d, doc.T(" "))
	}
	d = doc.Concat(d, p.typeParams(tparams), p.params(params), p.returnType(ret))
	if body != nil {
		d = doc.Concat(d, doc.T(" "), p.node(body))
	}
	return d
}

func (p *Printer) class(n *ast.Node) *doc.Doc {
	var name *ast.Node
	var tparams, heritage, members *ast.NodeList
	kw := "class"
	if n.Kind == ast.KindClassDeclaration {
		c := n.AsClassDeclaration()
		name = c.Name()
		tparams = c.TypeParameters
		heritage = c.HeritageClauses
		members = c.Members
	} else {
		c := n.AsClassExpression()
		name = c.Name()
		tparams = c.TypeParameters
		heritage = c.HeritageClauses
		members = c.Members
	}
	d := doc.Concat(p.modifiers(n), doc.T(kw))
	if name != nil {
		d = doc.Concat(d, doc.T(" "), p.node(name))
	}
	d = doc.Concat(d, p.typeParams(tparams))
	if heritage != nil && len(heritage.Nodes) > 0 {
		var hs []*doc.Doc
		for _, h := range heritage.Nodes {
			hs = append(hs, p.node(h))
		}
		d = doc.Concat(d, doc.T(" "), doc.Join(doc.T(" "), hs))
	}
	return doc.Concat(d, doc.T(" "), p.classBody(members, n.End()))
}

func (p *Printer) classBody(members *ast.NodeList, end int) *doc.Doc {
	nodes := listNodes(members)
	var kept []*ast.Node
	for _, n := range nodes {
		if n != nil && n.Kind != ast.KindSemicolonClassElement && !ast.NodeIsMissing(n) {
			kept = append(kept, n)
		}
	}
	if len(kept) == 0 {
		dangle := p.commentLines(p.takeUntil(end))
		if dangle == nil {
			return doc.T("{}")
		}
		return doc.Group(doc.Concat(doc.T("{"), doc.Indent(doc.Concat(doc.HardLine(), dangle)), doc.HardLine(), doc.T("}")))
	}
	var parts []*doc.Doc
	for i, n := range kept {
		if i > 0 {
			parts = append(parts, doc.HardLine())
			if blankBetween(p.src, kept[i-1].End(), p.skip(n)) {
				parts = append(parts, doc.HardLine())
			}
		}
		parts = append(parts, p.finish(n, nextStmt(kept, i+1)))
	}
	if dangle := p.commentLines(p.takeUntil(end)); dangle != nil {
		parts = append(parts, doc.HardLine(), dangle)
	}
	return doc.Group(doc.Concat(doc.T("{"), doc.Indent(doc.Concat(doc.HardLine(), doc.Concat(parts...))), doc.HardLine(), doc.T("}")))
}

func (p *Printer) iface(n *ast.Node) *doc.Doc {
	i := n.AsInterfaceDeclaration()
	d := doc.Concat(p.modifiers(n), doc.T("interface "), p.node(i.Name()), p.typeParams(i.TypeParameters))
	if i.HeritageClauses != nil && len(i.HeritageClauses.Nodes) > 0 {
		var hs []*doc.Doc
		for _, h := range i.HeritageClauses.Nodes {
			hs = append(hs, p.node(h))
		}
		d = doc.Concat(d, doc.T(" "), doc.Join(doc.T(" "), hs))
	}
	return doc.Concat(d, doc.T(" "), p.memberList(i.Members, n.End(), true))
}

func (p *Printer) typeAlias(n *ast.Node) *doc.Doc {
	t := typeAliasData(n)
	header := doc.Concat(
		p.modifiers(n),
		doc.T("type "),
		p.node(t.Name()),
		p.typeParams(t.TypeParameters),
	)
	if t.Type != nil && t.Type.Kind == ast.KindIntersectionType {
		return doc.Group(doc.Concat(header, doc.T(" = "), p.intersectionParts(t.Type.AsIntersectionTypeNode().Types)))
	}
	var body *doc.Doc
	if t.Type != nil && t.Type.Kind == ast.KindUnionType {
		body = p.unionParts(t.Type.AsUnionTypeNode().Types, "|")
	} else {
		body = p.node(t.Type)
	}
	if t.Type != nil && hugValue(t.Type) {
		return doc.Group(doc.Concat(header, doc.T(" = "), body))
	}
	return doc.Group(doc.Concat(header, doc.T(" ="), doc.Indent(doc.Concat(doc.Line(), body))))
}

func (p *Printer) enum(n *ast.Node) *doc.Doc {
	e := n.AsEnumDeclaration()
	return doc.Concat(p.modifiers(n), doc.T("enum "), p.node(e.Name()), doc.T(" "), p.enumBody(e.Members, n.End()))
}

func (p *Printer) enumBody(members *ast.NodeList, end int) *doc.Doc {
	nodes := listNodes(members)
	return p.commaGroupBreak("{", "}", nodes, end, opt.TrailObject, p.opt.BracketSpacing, len(nodes) > 0, false)
}

func (p *Printer) enumMember(n *ast.Node) *doc.Doc {
	e := n.AsEnumMember()
	if e.Initializer == nil {
		return p.propName(e.Name())
	}
	return p.withValue(p.propName(e.Name()), " = ", " =", e.Initializer)
}

func (p *Printer) module(n *ast.Node) *doc.Doc {
	m := n.AsModuleDeclaration()
	kw := scanner.TokenToString(m.Keyword)
	if kw == "" {
		kw = "namespace"
	}
	d := doc.Concat(p.modifiers(n), doc.T(kw), doc.T(" "), p.node(m.Name()))
	if m.Body == nil {
		return d
	}
	if m.Body.Kind == ast.KindModuleDeclaration {
		// namespace A.B {} is nested. Print "namespace A.B".
		return doc.Concat(d, doc.T("."), p.nestedModule(m.Body))
	}
	return doc.Concat(d, doc.T(" "), p.node(m.Body))
}

func (p *Printer) nestedModule(n *ast.Node) *doc.Doc {
	m := n.AsModuleDeclaration()
	d := p.node(m.Name())
	if m.Body == nil {
		return d
	}
	if m.Body.Kind == ast.KindModuleDeclaration {
		return doc.Concat(d, doc.T("."), p.nestedModule(m.Body))
	}
	return doc.Concat(d, doc.T(" "), p.node(m.Body))
}

func (p *Printer) heritage(n *ast.Node) *doc.Doc {
	h := n.AsHeritageClause()
	kw := scanner.TokenToString(h.Token)
	if kw == "" {
		kw = "extends"
	}
	els := listNodes(h.Types)
	var parts []*doc.Doc
	for _, e := range els {
		parts = append(parts, p.heritageType(e))
	}
	return doc.Concat(doc.T(kw), doc.T(" "), doc.Join(doc.Concat(doc.T(","), doc.Line()), parts))
}

// heritageType parenthesizes a broken superclass so a long member chain stays
// grouped: `extends (\n  a.b.c\n)`.
func (p *Printer) heritageType(n *ast.Node) *doc.Doc {
	if n == nil || n.Kind != ast.KindExpressionWithTypeArguments {
		return p.node(n)
	}
	e := n.AsExpressionWithTypeArguments()
	expr := e.Expression
	body := p.node(expr)
	simple := expr == nil || expr.Kind == ast.KindIdentifier || expr.Kind == ast.KindThisKeyword
	if !simple {
		body = doc.Group(doc.Concat(
			doc.IfBreak(doc.T("("), nil),
			doc.Indent(doc.Concat(doc.SoftLine(), body)),
			doc.SoftLine(),
			doc.IfBreak(doc.T(")"), nil),
		))
	}
	return doc.Concat(body, p.typeArgs(e.TypeArguments))
}

func (p *Printer) importDecl(n *ast.Node) *doc.Doc {
	d := importData(n)
	if d.ImportClause == nil {
		return doc.Concat(p.modifiers(n), doc.T("import "), p.node(d.ModuleSpecifier), p.importAttrsDoc(d.Attributes))
	}
	return doc.Concat(p.modifiers(n), doc.T("import "), p.node(d.ImportClause), doc.T(" from "), p.node(d.ModuleSpecifier), p.importAttrsDoc(d.Attributes))
}

func (p *Printer) importAttrsDoc(n *ast.Node) *doc.Doc {
	if n == nil {
		return nil
	}
	return doc.Concat(doc.T(" "), p.node(n))
}

func (p *Printer) importClause(n *ast.Node) *doc.Doc {
	c := n.AsImportClause()
	var parts []*doc.Doc
	if c.PhaseModifier != ast.KindUnknown && c.PhaseModifier != 0 {
		if s := scanner.TokenToString(c.PhaseModifier); s != "" {
			parts = append(parts, doc.T(s), doc.T(" "))
		}
	}
	if c.Name() != nil {
		parts = append(parts, p.node(c.Name()))
		if c.NamedBindings != nil {
			parts = append(parts, doc.T(", "))
		}
	}
	if c.NamedBindings != nil {
		parts = append(parts, p.node(c.NamedBindings))
	}
	return doc.Concat(parts...)
}

func (p *Printer) namedList(n *ast.Node) *doc.Doc {
	var els []*ast.Node
	end := n.End()
	if n.Kind == ast.KindNamedImports {
		els = listNodes(n.AsNamedImports().Elements)
	} else {
		els = listNodes(n.AsNamedExports().Elements)
	}
	return p.commaGroup("{", "}", els, end, opt.TrailImport, p.opt.BracketSpacing)
}

func (p *Printer) importSpec(n *ast.Node) *doc.Doc {
	s := n.AsImportSpecifier()
	var d *doc.Doc
	if s.IsTypeOnly {
		d = doc.T("type ")
	}
	if s.PropertyName != nil {
		return doc.Concat(d, p.node(s.PropertyName), doc.T(" as "), p.node(s.Name()))
	}
	return doc.Concat(d, p.node(s.Name()))
}

func (p *Printer) exportSpec(n *ast.Node) *doc.Doc {
	s := n.AsExportSpecifier()
	var d *doc.Doc
	if s.IsTypeOnly {
		d = doc.T("type ")
	}
	if s.PropertyName != nil {
		return doc.Concat(d, p.node(s.PropertyName), doc.T(" as "), p.node(s.Name()))
	}
	return doc.Concat(d, p.node(s.Name()))
}

func (p *Printer) importEquals(n *ast.Node) *doc.Doc {
	d := n.AsImportEqualsDeclaration()
	t := ""
	if d.IsTypeOnly {
		t = "type "
	}
	return doc.Concat(p.modifiers(n), doc.T("import "), doc.T(t), p.node(d.Name()), doc.T(" = "), p.node(d.ModuleReference))
}

func (p *Printer) exportAssign(n *ast.Node) *doc.Doc {
	e := n.AsExportAssignment()
	if e.IsExportEquals {
		return doc.Concat(p.modifiers(n), doc.T("export = "), p.node(e.Expression))
	}
	return doc.Concat(p.modifiers(n), doc.T("export default "), p.node(e.Expression))
}

func (p *Printer) exportDecl(n *ast.Node) *doc.Doc {
	e := n.AsExportDeclaration()
	d := doc.Concat(p.modifiers(n), doc.T("export "))
	if e.IsTypeOnly {
		d = doc.Concat(d, doc.T("type "))
	}
	if e.ExportClause == nil {
		d = doc.Concat(d, doc.T("*"))
	} else {
		d = doc.Concat(d, p.node(e.ExportClause))
	}
	if e.ModuleSpecifier != nil {
		d = doc.Concat(d, doc.T(" from "), p.node(e.ModuleSpecifier))
	}
	return doc.Concat(d, p.importAttrsDoc(e.Attributes))
}

func (p *Printer) importAttrs(n *ast.Node) *doc.Doc {
	a := n.AsImportAttributes()
	kw := scanner.TokenToString(a.Token)
	if kw == "" {
		kw = "with"
	}
	return doc.Concat(doc.T(kw), doc.T(" "), p.commaGroup("{", "}", listNodes(a.Attributes), n.End(), opt.TrailObject, true))
}

func (p *Printer) member(n *ast.Node) *doc.Doc {
	switch n.Kind {
	case ast.KindSemicolonClassElement:
		return nil
	case ast.KindClassStaticBlockDeclaration:
		b := n.AsClassStaticBlockDeclaration()
		return doc.Concat(doc.T("static "), p.node(b.Body))
	case ast.KindConstructor:
		c := n.AsConstructorDeclaration()
		d := doc.Concat(p.modifiers(n), doc.T("constructor"), p.typeParams(c.TypeParameters), p.params(c.Parameters), p.returnType(c.Type))
		if c.Body != nil {
			d = doc.Concat(d, doc.T(" "), p.node(c.Body))
		}
		return d
	case ast.KindMethodDeclaration:
		m := n.AsMethodDeclaration()
		return p.methodLike(n, m.AsteriskToken, m.Name(), m.PostfixToken, m.TypeParameters, m.Parameters, m.Type, m.Body, "")
	case ast.KindMethodSignature:
		m := n.AsMethodSignatureDeclaration()
		return p.methodLike(n, nil, m.Name(), m.PostfixToken, m.TypeParameters, m.Parameters, m.Type, nil, "")
	case ast.KindGetAccessor:
		m := n.AsGetAccessorDeclaration()
		return p.methodLike(n, nil, m.Name(), nil, m.TypeParameters, m.Parameters, m.Type, m.Body, "get")
	case ast.KindSetAccessor:
		m := n.AsSetAccessorDeclaration()
		return p.methodLike(n, nil, m.Name(), nil, m.TypeParameters, m.Parameters, m.Type, m.Body, "set")
	case ast.KindPropertyDeclaration:
		m := n.AsPropertyDeclaration()
		return p.propDecl(n, m.Name(), m.PostfixToken, m.Type, m.Initializer)
	case ast.KindPropertySignature:
		m := n.AsPropertySignatureDeclaration()
		return p.propDecl(n, m.Name(), m.PostfixToken, m.Type, m.Initializer)
	case ast.KindCallSignature:
		c := n.AsCallSignatureDeclaration()
		return doc.Concat(p.typeParams(c.TypeParameters), p.params(c.Parameters), p.returnType(c.Type))
	case ast.KindConstructSignature:
		c := n.AsConstructSignatureDeclaration()
		return doc.Concat(p.modifiers(n), doc.T("new "), p.typeParams(c.TypeParameters), p.params(c.Parameters), p.returnType(c.Type))
	case ast.KindIndexSignature:
		ix := n.AsIndexSignatureDeclaration()
		return doc.Concat(p.modifiers(n), p.bracketParams(ix.Parameters), p.returnType(ix.Type))
	default:
		return p.rawFallback(n)
	}
}

func (p *Printer) methodLike(n *ast.Node, star *ast.Node, name *ast.Node, postfix *ast.Node, tparams, params *ast.NodeList, ret, body *ast.Node, word string) *doc.Doc {
	d := p.modifiers(n)
	if word != "" {
		d = doc.Concat(d, doc.T(word), doc.T(" "))
	}
	if star != nil {
		d = doc.Concat(d, doc.T("*"))
	}
	d = doc.Concat(d, p.propName(name))
	if postfix != nil {
		d = doc.Concat(d, p.node(postfix))
	}
	d = doc.Concat(d, p.typeParams(tparams), p.params(params), p.returnType(ret))
	if body != nil {
		d = doc.Concat(d, doc.T(" "), p.node(body))
	}
	return d
}

func (p *Printer) propDecl(n *ast.Node, name, postfix, typ, init *ast.Node) *doc.Doc {
	d := doc.Concat(p.modifiers(n), p.propName(name))
	if postfix != nil {
		d = doc.Concat(d, p.node(postfix))
	}
	return p.nameInit(d, false, typ, init)
}

func (p *Printer) memberList(list *ast.NodeList, end int, expand bool) *doc.Doc {
	nodes := listNodes(list)
	if len(nodes) == 0 {
		if dangle := p.commentLines(p.takeUntil(end)); dangle != nil {
			return doc.Group(doc.Concat(doc.T("{"), doc.Indent(doc.Concat(doc.HardLine(), dangle)), doc.HardLine(), doc.T("}")))
		}
		return doc.T("{}")
	}
	var parts []*doc.Doc
	for _, n := range nodes {
		if n == nil || ast.NodeIsMissing(n) {
			continue
		}
		parts = append(parts, p.node(n))
	}
	sep := doc.Line()
	if p.opt.Semi {
		sep = doc.Concat(doc.T(";"), doc.Line())
	}
	body := doc.Join(sep, parts)
	if p.opt.Semi {
		body = doc.Concat(body, doc.IfBreak(doc.T(";"), nil))
	}
	openBreak := doc.SoftLine()
	if p.opt.BracketSpacing {
		openBreak = doc.Line()
	}
	dangle := p.commentLines(p.takeUntil(end))
	inner := doc.Concat(openBreak, body)
	force := expand
	if !force {
		for _, part := range parts {
			if doc.ForcesBreak(part) {
				force = true
				break
			}
		}
	}
	if dangle != nil {
		inner = doc.Concat(inner, doc.HardLine(), dangle)
		force = true
	}
	grouped := doc.Concat(doc.T("{"), doc.Indent(inner), openBreak, doc.T("}"))
	if force {
		return doc.BreakGroup(grouped)
	}
	return doc.Group(grouped)
}

func (p *Printer) parameter(n *ast.Node) *doc.Doc {
	param := n.AsParameterDeclaration()
	var d *doc.Doc
	d = p.modifiers(n)
	if param.DotDotDotToken != nil {
		d = doc.Concat(d, doc.T("..."))
	}
	d = doc.Concat(d, p.binding(param.Name()))
	if param.QuestionToken != nil {
		d = doc.Concat(d, doc.T("?"))
	}
	return p.nameInit(d, false, param.Type, param.Initializer)
}

func (p *Printer) params(list *ast.NodeList) *doc.Doc {
	if list == nil {
		return doc.T("()")
	}
	end := p.skipCloser(list.End(), ')')
	return p.commaGroup("(", ")", list.Nodes, end, opt.TrailParams, false)
}

func (p *Printer) bracketParams(list *ast.NodeList) *doc.Doc {
	return p.commaGroup("[", "]", listNodes(list), listEnd(list), opt.TrailNone, false)
}

func listEnd(list *ast.NodeList) int {
	if list == nil {
		return 0
	}
	return list.End()
}

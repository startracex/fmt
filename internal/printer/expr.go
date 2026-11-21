package printer

import (
	"strings"

	"github.com/startracex/tsgo/pkg/ast"
	"github.com/startracex/tsgo/pkg/scanner"

	"github.com/startracex/fmt/doc"
	"github.com/startracex/fmt/internal/opt"
)

func (p *Printer) array(n *ast.Node) *doc.Doc {
	a := n.AsArrayLiteralExpression()
	els := listNodes(a.Elements)
	return p.commaGroupBreak("[", "]", els, n.End(), opt.TrailArray, false, p.opensBroken(n, '[', els))
}

func (p *Printer) object(n *ast.Node) *doc.Doc {
	o := n.AsObjectLiteralExpression()
	if p.opt.QuoteProps == "consistent" {
		p.forceQuote = objectNeedsQuotes(o.Properties)
		defer func() { p.forceQuote = false }()
	}
	props := listNodes(o.Properties)
	return p.commaGroupBreak("{", "}", props, n.End(), opt.TrailObject, p.opt.BracketSpacing, p.opensBroken(n, '{', props))
}

// opensBroken reports whether the source has a line break between the opening
// brace or bracket and its first element. Prettier's default objectWrap
// preserve keeps that collection expanded. Node positions point at leading
// trivia, so the newline is often the first byte of the child.
func (p *Printer) opensBroken(n *ast.Node, open byte, elems []*ast.Node) bool {
	first := firstReal(elems)
	if n == nil || first == nil {
		return false
	}
	start := n.Pos()
	end := first.Pos()
	if start < 0 || end < start || start >= len(p.src) {
		return false
	}
	limit := end + 1
	if limit > len(p.src) {
		limit = len(p.src)
	}
	chunk := p.src[start:limit]
	brace := strings.IndexByte(chunk, open)
	if brace < 0 {
		return strings.Contains(chunk, "\n")
	}
	return strings.Contains(chunk[brace:], "\n")
}

func firstReal(nodes []*ast.Node) *ast.Node {
	for _, n := range nodes {
		if n != nil && !ast.NodeIsMissing(n) {
			return n
		}
	}
	return nil
}

func objectNeedsQuotes(list *ast.NodeList) bool {
	if list == nil {
		return false
	}
	for _, n := range list.Nodes {
		name := propertyNameOf(n)
		if name == nil || name.Kind != ast.KindStringLiteral {
			continue
		}
		if !scanner.IsValidIdentifier(name.AsStringLiteral().Text) {
			return true
		}
	}
	return false
}

func propertyNameOf(n *ast.Node) *ast.Node {
	if n == nil {
		return nil
	}
	switch n.Kind {
	case ast.KindPropertyAssignment:
		return n.AsPropertyAssignment().Name()
	case ast.KindShorthandPropertyAssignment:
		return n.AsShorthandPropertyAssignment().Name()
	case ast.KindMethodDeclaration:
		return n.AsMethodDeclaration().Name()
	case ast.KindGetAccessor:
		return n.AsGetAccessorDeclaration().Name()
	case ast.KindSetAccessor:
		return n.AsSetAccessorDeclaration().Name()
	default:
		return nil
	}
}

func (p *Printer) commaGroup(open, close string, nodes []*ast.Node, end int, trail opt.TrailKind, bracketSpace bool) *doc.Doc {
	return p.commaGroupBreak(open, close, nodes, end, trail, bracketSpace, false)
}

func (p *Printer) commaGroupBreak(open, close string, nodes []*ast.Node, end int, trail opt.TrailKind, bracketSpace, force bool) *doc.Doc {
	var elems []*doc.Doc
	for _, n := range nodes {
		if n == nil || ast.NodeIsMissing(n) {
			continue
		}
		elems = append(elems, p.node(n))
	}
	dangle := p.commentLines(p.takeUntil(end))
	if len(elems) == 0 && dangle == nil {
		return doc.T(open + close)
	}
	var body *doc.Doc
	if len(elems) > 0 {
		if !force {
			for _, el := range elems {
				if doc.ContainsHard(el) {
					force = true
					break
				}
			}
		}
		body = doc.Join(doc.Concat(doc.T(","), doc.Line()), elems)
		if p.opt.Trailing(trail) {
			body = doc.Concat(body, doc.IfBreak(doc.T(","), nil))
		}
	}
	breakDoc := doc.SoftLine()
	if bracketSpace {
		breakDoc = doc.Line()
	}
	inner := doc.Concat(breakDoc, body)
	if dangle != nil {
		if body != nil {
			inner = doc.Concat(inner, doc.HardLine(), dangle)
		} else {
			inner = doc.Concat(doc.HardLine(), dangle, doc.HardLine())
		}
		force = true
	}
	grouped := doc.Concat(doc.T(open), doc.Indent(inner), breakDoc, doc.T(close))
	if force {
		return doc.BreakGroup(grouped)
	}
	return doc.Group(grouped)
}

func (p *Printer) propAssign(n *ast.Node) *doc.Doc {
	a := n.AsPropertyAssignment()
	return doc.Group(doc.Concat(
		p.modifiers(n),
		p.propName(a.Name()),
		doc.T(":"),
		doc.Group(doc.Indent(doc.Concat(doc.Line(), p.node(a.Initializer)))),
	))
}

func (p *Printer) shorthand(n *ast.Node) *doc.Doc {
	s := n.AsShorthandPropertyAssignment()
	d := doc.Concat(p.modifiers(n), p.propName(s.Name()))
	if s.ObjectAssignmentInitializer != nil {
		d = doc.Group(doc.Concat(d, doc.T(" ="), doc.Indent(doc.Concat(doc.Line(), p.node(s.ObjectAssignmentInitializer)))))
	}
	return d
}

func (p *Printer) spreadExpr(n *ast.Node) *ast.Node {
	if n.Kind == ast.KindSpreadElement {
		return n.AsSpreadElement().Expression
	}
	return n.AsSpreadAssignment().Expression
}

func (p *Printer) template(n *ast.Node) *doc.Doc {
	t := n.AsTemplateExpression()
	parts := []*doc.Doc{doc.T(p.raw(t.Head))}
	if t.TemplateSpans != nil {
		for _, sp := range t.TemplateSpans.Nodes {
			s := sp.AsTemplateSpan()
			parts = append(parts, p.node(s.Expression), doc.T(p.raw(s.Literal)))
		}
	}
	return doc.Concat(parts...)
}

func (p *Printer) chain(n *ast.Node) *doc.Doc {
	var links []*ast.Node
	cur := n
	for {
		next := chainTarget(cur)
		if next == nil {
			break
		}
		links = append(links, cur)
		cur = next
	}
	for i, j := 0, len(links)-1; i < j; i, j = i+1, j-1 {
		links[i], links[j] = links[j], links[i]
	}
	head := p.node(cur)
	if len(links) == 0 {
		return head
	}
	var segs []*doc.Doc
	indent := false
	for _, l := range links {
		if chainBreakBefore(l) {
			segs = append(segs, doc.SoftLine())
			indent = true
		}
		segs = append(segs, p.chainSuffix(l))
	}
	body := doc.Concat(segs...)
	// A bare call has no break of its own. Indenting it would add a second
	// level to the argument object when the call does not fit.
	if indent {
		body = doc.Indent(body)
	}
	return doc.Group(doc.Concat(head, body))
}

func chainTarget(n *ast.Node) *ast.Node {
	switch n.Kind {
	case ast.KindCallExpression:
		return n.AsCallExpression().Expression
	case ast.KindPropertyAccessExpression:
		return n.AsPropertyAccessExpression().Expression
	case ast.KindElementAccessExpression:
		return n.AsElementAccessExpression().Expression
	case ast.KindTaggedTemplateExpression:
		return n.AsTaggedTemplateExpression().Tag
	default:
		return nil
	}
}

func chainBreakBefore(n *ast.Node) bool {
	switch n.Kind {
	case ast.KindPropertyAccessExpression, ast.KindElementAccessExpression:
		return true
	case ast.KindCallExpression:
		return n.AsCallExpression().QuestionDotToken != nil
	default:
		return false
	}
}

func (p *Printer) chainSuffix(n *ast.Node) *doc.Doc {
	switch n.Kind {
	case ast.KindPropertyAccessExpression:
		e := n.AsPropertyAccessExpression()
		dot := "."
		if e.QuestionDotToken != nil {
			dot = "?."
		}
		return doc.Concat(doc.T(dot), p.node(e.Name()))
	case ast.KindElementAccessExpression:
		e := n.AsElementAccessExpression()
		dot := ""
		if e.QuestionDotToken != nil {
			dot = "?."
		}
		return doc.Concat(doc.T(dot), doc.T("["), p.node(e.ArgumentExpression), p.beforeClose(n.End()), doc.T("]"))
	case ast.KindCallExpression:
		c := n.AsCallExpression()
		var dot *doc.Doc
		if c.QuestionDotToken != nil {
			dot = doc.T("?.")
		}
		return doc.Concat(dot, p.typeArgs(c.TypeArguments), p.callArgs(c.Arguments, n.End()))
	case ast.KindTaggedTemplateExpression:
		t := n.AsTaggedTemplateExpression()
		return doc.Concat(p.typeArgs(t.TypeArguments), p.node(t.Template))
	default:
		return p.node(n)
	}
}

func (p *Printer) beforeClose(end int) *doc.Doc {
	return p.inlineComments(p.takeUntil(end))
}

func (p *Printer) callArgs(list *ast.NodeList, end int) *doc.Doc {
	nodes := realNodes(listNodes(list))
	if len(nodes) == 0 {
		return doc.Concat(doc.T("("), p.beforeClose(end), doc.T(")"))
	}
	last := nodes[len(nodes)-1]
	// A trailing object or array stays against the call: defineConfig({ ... }).
	// Two collections of the same kind still break like an ordinary list.
	if huggable(last) && (len(nodes) == 1 || !sameCollection(nodes[len(nodes)-2], last)) {
		return p.hugArgs(nodes, end)
	}
	return p.commaGroup("(", ")", nodes, end, opt.TrailArgs, false)
}

func (p *Printer) hugArgs(nodes []*ast.Node, end int) *doc.Doc {
	if len(nodes) == 1 {
		return doc.Concat(doc.T("("), p.node(nodes[0]), p.beforeClose(end), doc.T(")"))
	}
	head := p.argJoin(nodes[:len(nodes)-1])
	last := p.node(nodes[len(nodes)-1])
	close := doc.Concat(p.beforeClose(end), doc.T(")"))
	return doc.Group(doc.Concat(
		doc.T("("),
		doc.Indent(doc.Concat(
			doc.SoftLine(),
			head,
			doc.T(","),
			doc.Line(),
			last,
		)),
		doc.IfBreak(doc.T(","), nil),
		doc.SoftLine(),
		close,
	))
}

func (p *Printer) argJoin(nodes []*ast.Node) *doc.Doc {
	parts := make([]*doc.Doc, 0, len(nodes))
	for _, n := range nodes {
		parts = append(parts, p.node(n))
	}
	return doc.Join(doc.Concat(doc.T(","), doc.Line()), parts)
}

func huggable(n *ast.Node) bool {
	if n == nil {
		return false
	}
	switch n.Kind {
	case ast.KindObjectLiteralExpression, ast.KindArrayLiteralExpression:
		return true
	default:
		return false
	}
}

func sameCollection(a, b *ast.Node) bool {
	return a != nil && b != nil && a.Kind == b.Kind && huggable(a)
}

func realNodes(nodes []*ast.Node) []*ast.Node {
	out := make([]*ast.Node, 0, len(nodes))
	for _, n := range nodes {
		if n != nil && !ast.NodeIsMissing(n) {
			out = append(out, n)
		}
	}
	return out
}

func (p *Printer) newExpr(n *ast.Node) *doc.Doc {
	e := n.AsNewExpression()
	d := doc.Concat(doc.T("new "), p.node(e.Expression), p.typeArgs(e.TypeArguments))
	if e.Arguments != nil {
		d = doc.Concat(d, p.callArgs(e.Arguments, n.End()))
	}
	return d
}

func (p *Printer) prefix(n *ast.Node) *doc.Doc {
	u := n.AsPrefixUnaryExpression()
	op := scanner.TokenToString(u.Operator)
	if op == "" {
		op = p.raw(n)[:1]
	}
	space := ""
	if wordOp(u.Operator) {
		space = " "
	}
	return doc.Concat(doc.T(op), doc.T(space), p.node(u.Operand))
}

func (p *Printer) postfix(n *ast.Node) *doc.Doc {
	u := n.AsPostfixUnaryExpression()
	op := scanner.TokenToString(u.Operator)
	return doc.Concat(p.node(u.Operand), doc.T(op))
}

func (p *Printer) wordUnary(n *ast.Node) *doc.Doc {
	kw := scanner.TokenToString(keywordOfUnary(n.Kind))
	return doc.Concat(doc.T(kw), doc.T(" "), p.node(n.Expression()))
}

func keywordOfUnary(k ast.Kind) ast.Kind {
	switch k {
	case ast.KindDeleteExpression:
		return ast.KindDeleteKeyword
	case ast.KindTypeOfExpression:
		return ast.KindTypeOfKeyword
	case ast.KindVoidExpression:
		return ast.KindVoidKeyword
	case ast.KindAwaitExpression:
		return ast.KindAwaitKeyword
	default:
		return k
	}
}

func wordOp(k ast.Kind) bool {
	switch k {
	case ast.KindDeleteKeyword, ast.KindTypeOfKeyword, ast.KindVoidKeyword, ast.KindAwaitKeyword:
		return true
	default:
		return false
	}
}

func (p *Printer) yield(n *ast.Node) *doc.Doc {
	y := n.AsYieldExpression()
	kw := "yield"
	if y.AsteriskToken != nil {
		kw = "yield*"
	}
	if y.Expression == nil {
		return doc.T(kw)
	}
	return doc.Group(doc.Concat(doc.T(kw), doc.Indent(doc.Concat(doc.Line(), p.node(y.Expression)))))
}

func (p *Printer) conditional(n *ast.Node, isType bool) *doc.Doc {
	var test, a, b *doc.Doc
	var opA, opB string
	if isType {
		c := n.AsConditionalTypeNode()
		test = doc.Concat(p.node(c.CheckType), doc.T(" extends "), p.node(c.ExtendsType))
		a = p.node(c.TrueType)
		b = p.node(c.FalseType)
		opA, opB = "? ", ": "
	} else {
		c := n.AsConditionalExpression()
		test = p.node(c.Condition)
		a = p.node(c.WhenTrue)
		b = p.node(c.WhenFalse)
		opA, opB = "? ", ": "
	}
	return doc.Group(doc.Concat(
		test,
		doc.Indent(doc.Concat(
			doc.Line(), doc.T(opA), a,
			doc.Line(), doc.T(opB), b,
		)),
	))
}

func (p *Printer) binary(n *ast.Node) *doc.Doc {
	b := n.AsBinaryExpression()
	op := b.OperatorToken.Kind
	if op == ast.KindCommaToken {
		return p.sequence(n)
	}
	if isAssign(op) {
		return doc.Group(doc.Concat(
			p.node(b.Left),
			doc.T(" "),
			doc.T(scanner.TokenToString(op)),
			doc.Group(doc.Indent(doc.Concat(doc.Line(), p.node(b.Right)))),
		))
	}
	ops, operands := p.binaryDocs(n, prec(op))
	if len(operands) == 0 {
		return nil
	}
	if len(ops) == 0 {
		return operands[0]
	}
	// A op
	//   B op
	//   C
	d := doc.Concat(operands[0], doc.T(" "), doc.T(ops[0]))
	var rest []*doc.Doc
	for i := 1; i < len(operands); i++ {
		rest = append(rest, doc.Line(), operands[i])
		if i < len(ops) {
			rest = append(rest, doc.T(" "), doc.T(ops[i]))
		}
	}
	return doc.Group(doc.Concat(d, doc.Indent(doc.Concat(rest...))))
}

func (p *Printer) sequence(n *ast.Node) *doc.Doc {
	var parts []*doc.Doc
	var walk func(*ast.Node)
	walk = func(n *ast.Node) {
		if n.Kind == ast.KindBinaryExpression && n.AsBinaryExpression().OperatorToken.Kind == ast.KindCommaToken {
			b := n.AsBinaryExpression()
			walk(b.Left)
			parts = append(parts, p.node(b.Right))
			return
		}
		parts = append(parts, p.node(n))
	}
	walk(n)
	return doc.Group(doc.Join(doc.Concat(doc.T(","), doc.Line()), parts))
}

func (p *Printer) binaryDocs(n *ast.Node, level int) (ops []string, operands []*doc.Doc) {
	var rights []struct {
		op   string
		node *ast.Node
	}
	cur := n
	for cur != nil && cur.Kind == ast.KindBinaryExpression {
		b := cur.AsBinaryExpression()
		op := b.OperatorToken.Kind
		if isAssign(op) || op == ast.KindCommaToken || prec(op) != level {
			break
		}
		rights = append(rights, struct {
			op   string
			node *ast.Node
		}{scanner.TokenToString(op), b.Right})
		cur = b.Left
	}
	operands = append(operands, p.node(cur))
	for i := len(rights) - 1; i >= 0; i-- {
		ops = append(ops, rights[i].op)
		operands = append(operands, p.node(rights[i].node))
	}
	return ops, operands
}

func isAssign(k ast.Kind) bool {
	return k >= ast.KindFirstAssignment && k <= ast.KindLastAssignment
}

func prec(k ast.Kind) int {
	switch k {
	case ast.KindCommaToken:
		return 1
	case ast.KindEqualsToken, ast.KindPlusEqualsToken, ast.KindMinusEqualsToken,
		ast.KindAsteriskEqualsToken, ast.KindAsteriskAsteriskEqualsToken, ast.KindSlashEqualsToken,
		ast.KindPercentEqualsToken, ast.KindLessThanLessThanEqualsToken, ast.KindGreaterThanGreaterThanEqualsToken,
		ast.KindGreaterThanGreaterThanGreaterThanEqualsToken, ast.KindAmpersandEqualsToken, ast.KindBarEqualsToken,
		ast.KindBarBarEqualsToken, ast.KindAmpersandAmpersandEqualsToken, ast.KindQuestionQuestionEqualsToken,
		ast.KindCaretEqualsToken:
		return 2
	case ast.KindQuestionQuestionToken:
		return 3
	case ast.KindBarBarToken:
		return 4
	case ast.KindAmpersandAmpersandToken:
		return 5
	case ast.KindBarToken:
		return 6
	case ast.KindCaretToken:
		return 7
	case ast.KindAmpersandToken:
		return 8
	case ast.KindEqualsEqualsToken, ast.KindEqualsEqualsEqualsToken, ast.KindExclamationEqualsToken, ast.KindExclamationEqualsEqualsToken:
		return 9
	case ast.KindLessThanToken, ast.KindGreaterThanToken, ast.KindLessThanEqualsToken, ast.KindGreaterThanEqualsToken,
		ast.KindInstanceOfKeyword, ast.KindInKeyword:
		return 10
	case ast.KindLessThanLessThanToken, ast.KindGreaterThanGreaterThanToken, ast.KindGreaterThanGreaterThanGreaterThanToken:
		return 11
	case ast.KindPlusToken, ast.KindMinusToken:
		return 12
	case ast.KindAsteriskToken, ast.KindSlashToken, ast.KindPercentToken:
		return 13
	case ast.KindAsteriskAsteriskToken:
		return 14
	default:
		return 0
	}
}

func (p *Printer) arrow(n *ast.Node) *doc.Doc {
	a := n.AsArrowFunction()
	head := p.modifiers(n)
	head = doc.Concat(head, p.typeParams(a.TypeParameters))
	if p.singleArrowParam(a) {
		head = doc.Concat(head, p.node(a.Parameters.Nodes[0]))
	} else {
		head = doc.Concat(head, p.params(a.Parameters))
	}
	head = doc.Concat(head, p.returnType(a.Type), doc.T(" =>"))
	if a.Body != nil && a.Body.Kind == ast.KindBlock {
		return doc.Concat(head, doc.T(" "), p.node(a.Body))
	}
	if jsx, ok := jsxInner(a.Body); ok {
		return doc.Concat(head, p.jsxBreak(" ", " (", jsx))
	}
	return doc.Group(doc.Concat(head, doc.Indent(doc.Concat(doc.Line(), p.node(a.Body)))))
}

func (p *Printer) singleArrowParam(a *ast.ArrowFunction) bool {
	if p.opt.ArrowParens != "avoid" || a.TypeParameters != nil || a.Type != nil || a.Parameters == nil || len(a.Parameters.Nodes) != 1 {
		return false
	}
	param := a.Parameters.Nodes[0]
	if param.Kind != ast.KindParameter {
		return false
	}
	pdecl := param.AsParameterDeclaration()
	if pdecl.DotDotDotToken != nil || pdecl.QuestionToken != nil || pdecl.Type != nil || pdecl.Initializer != nil {
		return false
	}
	if mods := param.Modifiers(); mods != nil && len(mods.Nodes) > 0 {
		return false
	}
	return pdecl.Name() != nil && pdecl.Name().Kind == ast.KindIdentifier
}

func quote(s string, single bool) string {
	q := byte('"')
	if single {
		q = '\''
	}
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte(q)
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\v':
			b.WriteString(`\v`)
		case '\'':
			if single {
				b.WriteString(`\'`)
			} else {
				b.WriteRune(r)
			}
		case '"':
			if !single {
				b.WriteString(`\"`)
			} else {
				b.WriteRune(r)
			}
		case '\u2028':
			b.WriteString(`\u2028`)
		case '\u2029':
			b.WriteString(`\u2029`)
		default:
			if r < 0x20 {
				b.WriteString(`\u`)
				const hex = "0123456789abcdef"
				b.WriteByte(hex[(r>>12)&0xF])
				b.WriteByte(hex[(r>>8)&0xF])
				b.WriteByte(hex[(r>>4)&0xF])
				b.WriteByte(hex[r&0xF])
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte(q)
	return b.String()
}

func useSingle(s string, preferSingle bool) bool {
	hasS := strings.Contains(s, "'")
	hasD := strings.Contains(s, "\"")
	if preferSingle {
		if hasS && !hasD {
			return false
		}
		return true
	}
	if hasD && !hasS {
		return true
	}
	return false
}

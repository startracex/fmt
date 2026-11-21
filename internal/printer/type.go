package printer

import (
	"github.com/startracex/tsgo/pkg/ast"
	"github.com/startracex/tsgo/pkg/scanner"

	"github.com/startracex/fmt/doc"
	"github.com/startracex/fmt/internal/opt"
)

func (p *Printer) typeParams(list *ast.NodeList) *doc.Doc {
	if list == nil || len(list.Nodes) == 0 {
		return nil
	}
	end := p.skipCloser(list.End(), '>')
	return p.commaGroup("<", ">", list.Nodes, end, opt.TrailType, false)
}

func (p *Printer) typeArgs(list *ast.NodeList) *doc.Doc {
	return p.typeParams(list)
}

func (p *Printer) returnType(n *ast.Node) *doc.Doc {
	if n == nil {
		return nil
	}
	return doc.Concat(doc.T(": "), p.node(n))
}

func (p *Printer) typeParam(n *ast.Node) *doc.Doc {
	t := n.AsTypeParameterDeclaration()
	d := doc.Concat(p.modifiers(n), p.node(t.Name()))
	if t.Constraint != nil {
		// Mapped type parameters are `K in T`; ordinary ones are `T extends U`.
		kw := " extends "
		if n.Parent != nil && n.Parent.Kind == ast.KindMappedType {
			kw = " in "
		}
		d = doc.Concat(d, doc.T(kw), p.node(t.Constraint))
	}
	if t.DefaultType != nil {
		d = doc.Concat(d, doc.T(" = "), p.node(t.DefaultType))
	}
	return d
}

func (p *Printer) typeNode(n *ast.Node) *doc.Doc {
	switch n.Kind {
	case ast.KindAnyKeyword, ast.KindBigIntKeyword, ast.KindBooleanKeyword, ast.KindIntrinsicKeyword,
		ast.KindNeverKeyword, ast.KindNumberKeyword, ast.KindObjectKeyword, ast.KindStringKeyword,
		ast.KindSymbolKeyword, ast.KindUndefinedKeyword, ast.KindUnknownKeyword, ast.KindVoidKeyword:
		return doc.T(scanner.TokenToString(n.Kind))
	case ast.KindThisType:
		return doc.T("this")
	case ast.KindLiteralType:
		return p.node(n.AsLiteralTypeNode().Literal)
	case ast.KindUnionType:
		return p.unionish(n.AsUnionTypeNode().Types, "|")
	case ast.KindIntersectionType:
		return p.unionish(n.AsIntersectionTypeNode().Types, "&")
	case ast.KindTypeReference:
		r := n.AsTypeReferenceNode()
		return doc.Concat(p.node(r.TypeName), p.typeArgs(r.TypeArguments))
	case ast.KindTypeQuery:
		q := n.AsTypeQueryNode()
		return doc.Concat(doc.T("typeof "), p.node(q.ExprName), p.typeArgs(q.TypeArguments))
	case ast.KindArrayType:
		return doc.Concat(p.node(n.AsArrayTypeNode().ElementType), doc.T("[]"))
	case ast.KindTupleType:
		return p.commaGroup("[", "]", listNodes(n.AsTupleTypeNode().Elements), n.End(), opt.TrailType, false)
	case ast.KindNamedTupleMember:
		return p.namedTuple(n)
	case ast.KindOptionalType:
		return doc.Concat(p.node(n.AsOptionalTypeNode().Type), doc.T("?"))
	case ast.KindRestType:
		return doc.Concat(doc.T("..."), p.node(n.AsRestTypeNode().Type))
	case ast.KindParenthesizedType:
		return doc.Concat(doc.T("("), p.node(n.AsParenthesizedTypeNode().Type), doc.T(")"))
	case ast.KindTypeOperator:
		o := n.AsTypeOperatorNode()
		kw := scanner.TokenToString(o.Operator)
		if kw == "" {
			kw = "keyof"
		}
		return doc.Concat(doc.T(kw), doc.T(" "), p.node(o.Type))
	case ast.KindIndexedAccessType:
		ix := n.AsIndexedAccessTypeNode()
		return doc.Concat(p.node(ix.ObjectType), doc.T("["), p.node(ix.IndexType), doc.T("]"))
	case ast.KindInferType:
		inf := n.AsInferTypeNode()
		return doc.Concat(doc.T("infer "), p.node(inf.TypeParameter))
	case ast.KindConditionalType:
		return p.conditional(n, true)
	case ast.KindTypeLiteral:
		return p.memberList(n.AsTypeLiteralNode().Members, n.End(), false)
	case ast.KindFunctionType:
		f := n.AsFunctionTypeNode()
		return doc.Group(doc.Concat(p.typeParams(f.TypeParameters), p.params(f.Parameters), doc.T(" => "), p.node(f.Type)))
	case ast.KindConstructorType:
		c := n.AsConstructorTypeNode()
		return doc.Group(doc.Concat(p.modifiers(n), doc.T("new "), p.typeParams(c.TypeParameters), p.params(c.Parameters), doc.T(" => "), p.node(c.Type)))
	case ast.KindTypePredicate:
		return p.typePredicate(n)
	case ast.KindMappedType:
		return p.mapped(n)
	case ast.KindTemplateLiteralType:
		return p.templateType(n)
	case ast.KindTemplateLiteralTypeSpan:
		s := n.AsTemplateLiteralTypeSpan()
		return doc.Concat(p.node(s.Type), p.node(s.Literal))
	case ast.KindImportType:
		return p.importType(n)
	default:
		return p.rawFallback(n)
	}
}

func (p *Printer) unionish(list *ast.NodeList, op string) *doc.Doc {
	if op == "&" {
		return doc.Group(p.intersectionParts(list))
	}
	return doc.Group(p.unionParts(list, op))
}

// unionParts breaks with whichever group encloses it. Type aliases use this
// directly so a union that does not fit beside `type Name =` expands to
// leading `|` lines instead of a long continuation.
func (p *Printer) unionParts(list *ast.NodeList, op string) *doc.Doc {
	nodes := listNodes(list)
	if len(nodes) == 0 {
		return nil
	}
	parts := []*doc.Doc{p.node(nodes[0])}
	for _, n := range nodes[1:] {
		parts = append(parts, doc.Line(), doc.T(op), doc.T(" "), p.node(n))
	}
	return doc.Concat(doc.IfBreak(doc.T(op+" "), nil), doc.Concat(parts...))
}

// intersectionParts breaks after `&`, so a long alias stays
// `type Foo = A &\n  B &\n  C` instead of leading operators.
func (p *Printer) intersectionParts(list *ast.NodeList) *doc.Doc {
	nodes := listNodes(list)
	if len(nodes) == 0 {
		return nil
	}
	d := p.node(nodes[0])
	if len(nodes) == 1 {
		return d
	}
	var cont []*doc.Doc
	for _, n := range nodes[1:] {
		cont = append(cont, doc.T(" &"), doc.Line(), p.node(n))
	}
	return doc.Concat(d, doc.Indent(doc.Concat(cont...)))
}

func (p *Printer) namedTuple(n *ast.Node) *doc.Doc {
	m := n.AsNamedTupleMember()
	var d *doc.Doc
	if m.DotDotDotToken != nil {
		d = doc.T("...")
	}
	d = doc.Concat(d, p.node(m.Name()))
	if m.QuestionToken != nil {
		d = doc.Concat(d, doc.T("?"))
	}
	return doc.Concat(d, doc.T(": "), p.node(m.Type))
}

func (p *Printer) typePredicate(n *ast.Node) *doc.Doc {
	t := n.AsTypePredicateNode()
	var d *doc.Doc
	if t.AssertsModifier != nil {
		d = doc.T("asserts ")
	}
	d = doc.Concat(d, p.node(t.ParameterName))
	if t.Type != nil {
		d = doc.Concat(d, doc.T(" is "), p.node(t.Type))
	}
	return d
}

func (p *Printer) mapped(n *ast.Node) *doc.Doc {
	m := n.AsMappedTypeNode()
	inner := doc.Concat(
		doc.T(mappedReadonly(m.ReadonlyToken)),
		doc.T("["),
		p.node(m.TypeParameter),
	)
	if m.NameType != nil {
		inner = doc.Concat(inner, doc.T(" as "), p.node(m.NameType))
	}
	inner = doc.Concat(inner, doc.T("]"), doc.T(mappedQuestion(m.QuestionToken)))
	if m.Type != nil {
		inner = doc.Concat(inner, doc.T(": "), p.node(m.Type))
	}
	if m.Members != nil && len(m.Members.Nodes) > 0 {
		inner = doc.Concat(inner, doc.T("; "), p.memberList(m.Members, n.End(), false))
	}
	return doc.Group(doc.Concat(doc.T("{ "), inner, doc.T(" }")))
}

func mappedReadonly(tok *ast.Node) string {
	if tok == nil {
		return ""
	}
	switch tok.Kind {
	case ast.KindPlusToken:
		return "+readonly "
	case ast.KindMinusToken:
		return "-readonly "
	default:
		return "readonly "
	}
}

func mappedQuestion(tok *ast.Node) string {
	if tok == nil {
		return ""
	}
	switch tok.Kind {
	case ast.KindPlusToken:
		return "+?"
	case ast.KindMinusToken:
		return "-?"
	default:
		return "?"
	}
}

func (p *Printer) templateType(n *ast.Node) *doc.Doc {
	t := n.AsTemplateLiteralTypeNode()
	parts := []*doc.Doc{doc.T(p.raw(t.Head))}
	if t.TemplateSpans != nil {
		for _, sp := range t.TemplateSpans.Nodes {
			s := sp.AsTemplateLiteralTypeSpan()
			parts = append(parts, p.node(s.Type), doc.T(p.raw(s.Literal)))
		}
	}
	return doc.Concat(parts...)
}

func (p *Printer) importType(n *ast.Node) *doc.Doc {
	t := n.AsImportTypeNode()
	d := doc.T("import(")
	if t.IsTypeOf {
		d = doc.T("import(typeof ")
	}
	d = doc.Concat(d, p.node(t.Argument))
	if t.Attributes != nil {
		d = doc.Concat(d, doc.T(", "), p.node(t.Attributes))
	}
	d = doc.Concat(d, doc.T(")"))
	if t.Qualifier != nil {
		d = doc.Concat(d, doc.T("."), p.node(t.Qualifier))
	}
	return doc.Concat(d, p.typeArgs(t.TypeArguments))
}

func (p *Printer) skipCloser(from int, close byte) int {
	if from < 0 {
		from = 0
	}
	i := from
	for i < len(p.src) {
		if i+1 < len(p.src) && p.src[i] == '/' && (p.src[i+1] == '/' || p.src[i+1] == '*') {
			if p.src[i+1] == '/' {
				for i < len(p.src) && p.src[i] != '\n' {
					i++
				}
				continue
			}
			i += 2
			for i+1 < len(p.src) && !(p.src[i] == '*' && p.src[i+1] == '/') {
				i++
			}
			if i+1 < len(p.src) {
				i += 2
			}
			continue
		}
		if p.src[i] == close {
			return i + 1
		}
		switch p.src[i] {
		case ' ', '\t', '\n', '\r', ',':
			i++
			continue
		default:
			return from
		}
	}
	return from
}

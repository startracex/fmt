package printer

import (
	"strings"
	"unicode"

	"github.com/startracex/tsgo/pkg/ast"

	"github.com/startracex/fmt/doc"
)

func (p *Printer) jsx(n *ast.Node) *doc.Doc {
	switch n.Kind {
	case ast.KindJsxElement:
		return p.jsxElement(n)
	case ast.KindJsxSelfClosingElement:
		return p.jsxOpen(n, true)
	case ast.KindJsxFragment:
		return p.jsxFragment(n)
	case ast.KindJsxOpeningElement:
		return p.jsxOpen(n, false)
	case ast.KindJsxClosingElement:
		return p.jsxClose(n)
	case ast.KindJsxText:
		return doc.T(collapseJSXText(n.AsJsxText().Text))
	case ast.KindJsxExpression:
		return p.jsxExpr(n)
	case ast.KindJsxAttribute:
		return p.jsxAttr(n)
	case ast.KindJsxSpreadAttribute:
		return doc.Concat(doc.T("{..."), p.node(n.AsJsxSpreadAttribute().Expression), doc.T("}"))
	case ast.KindJsxNamespacedName:
		j := n.AsJsxNamespacedName()
		return doc.Concat(p.node(j.Namespace), doc.T(":"), p.node(j.Name()))
	case ast.KindJsxAttributes:
		return p.jsxAttrs(n.AsJsxAttributes().Properties)
	default:
		return p.rawFallback(n)
	}
}

func (p *Printer) jsxElement(n *ast.Node) *doc.Doc {
	el := n.AsJsxElement()
	open := p.jsxOpen(el.OpeningElement, false)
	close := p.jsxClose(el.ClosingElement)
	parts := p.jsxParts(el.Children)
	if len(parts) == 0 {
		return doc.Concat(open, close)
	}
	// Element children always break, and the hard lines propagate so a parent
	// return or assignment wraps the tag in parentheses.
	if jsxHasElement(el.Children) {
		return doc.Concat(open, doc.Indent(doc.Concat(doc.HardLine(), doc.Join(doc.HardLine(), parts))), doc.HardLine(), close)
	}
	return doc.Group(doc.Concat(
		open,
		doc.Indent(doc.Concat(doc.SoftLine(), doc.Join(doc.SoftLine(), parts))),
		doc.SoftLine(),
		close,
	))
}

func (p *Printer) jsxFragment(n *ast.Node) *doc.Doc {
	f := n.AsJsxFragment()
	parts := p.jsxParts(f.Children)
	if len(parts) == 0 {
		return doc.T("<></>")
	}
	if jsxHasElement(f.Children) {
		return doc.Concat(doc.T("<>"), doc.Indent(doc.Concat(doc.HardLine(), doc.Join(doc.HardLine(), parts))), doc.HardLine(), doc.T("</>"))
	}
	return doc.Group(doc.Concat(
		doc.T("<>"),
		doc.Indent(doc.Concat(doc.SoftLine(), doc.Join(doc.SoftLine(), parts))),
		doc.SoftLine(),
		doc.T("</>"),
	))
}

func jsxHasElement(list *ast.NodeList) bool {
	if list == nil {
		return false
	}
	for _, c := range list.Nodes {
		if c == nil {
			continue
		}
		switch c.Kind {
		case ast.KindJsxElement, ast.KindJsxSelfClosingElement, ast.KindJsxFragment:
			return true
		}
	}
	return false
}

func (p *Printer) jsxParts(list *ast.NodeList) []*doc.Doc {
	if list == nil {
		return nil
	}
	var parts []*doc.Doc
	for _, c := range list.Nodes {
		if c == nil {
			continue
		}
		if c.Kind == ast.KindJsxText {
			text := c.AsJsxText().Text
			if strings.TrimSpace(text) == "" {
				continue
			}
			parts = append(parts, doc.T(collapseJSXText(text)))
			continue
		}
		parts = append(parts, p.node(c))
	}
	return parts
}

func collapseJSXText(s string) string {
	fields := strings.FieldsFunc(s, unicode.IsSpace)
	return strings.Join(fields, " ")
}

func (p *Printer) jsxOpen(n *ast.Node, self bool) *doc.Doc {
	var tag *ast.Node
	var targs *ast.NodeList
	var attrs *ast.Node
	if self {
		e := n.AsJsxSelfClosingElement()
		tag, targs, attrs = e.TagName, e.TypeArguments, e.Attributes
	} else {
		e := n.AsJsxOpeningElement()
		tag, targs, attrs = e.TagName, e.TypeArguments, e.Attributes
	}
	head := doc.Concat(doc.T("<"), p.jsxTag(tag), p.typeArgs(targs))
	var attrDoc *doc.Doc
	if attrs != nil {
		attrDoc = p.jsxAttrs(attrs.AsJsxAttributes().Properties)
	}
	if attrDoc == nil {
		if self {
			return doc.Concat(head, doc.T(" />"))
		}
		return doc.Concat(head, doc.T(">"))
	}
	end := ">"
	endBreak := doc.SoftLine()
	if self {
		// A space before `/>` when the tag fits; a break before it when it does not.
		end = "/>"
		endBreak = doc.Line()
	}
	return doc.Group(doc.Concat(
		head,
		doc.Indent(doc.Concat(doc.Line(), attrDoc)),
		endBreak,
		doc.T(end),
	))
}

func (p *Printer) jsxClose(n *ast.Node) *doc.Doc {
	if n == nil {
		return nil
	}
	return doc.Concat(doc.T("</"), p.jsxTag(n.AsJsxClosingElement().TagName), doc.T(">"))
}

func (p *Printer) jsxTag(n *ast.Node) *doc.Doc {
	if n == nil {
		return nil
	}
	switch n.Kind {
	case ast.KindPropertyAccessExpression:
		e := n.AsPropertyAccessExpression()
		return doc.Concat(p.jsxTag(e.Expression), doc.T("."), p.node(e.Name()))
	case ast.KindJsxNamespacedName:
		return p.jsx(n)
	default:
		return p.node(n)
	}
}

func (p *Printer) jsxAttrs(list *ast.NodeList) *doc.Doc {
	nodes := listNodes(list)
	if len(nodes) == 0 {
		return nil
	}
	parts := make([]*doc.Doc, 0, len(nodes))
	for _, n := range nodes {
		parts = append(parts, p.node(n))
	}
	return doc.Join(doc.Line(), parts)
}

func (p *Printer) jsxAttr(n *ast.Node) *doc.Doc {
	a := n.AsJsxAttribute()
	name := p.jsxTag(a.Name())
	if a.Initializer == nil {
		return name
	}
	return doc.Concat(name, doc.T("="), p.jsxAttrValue(a.Initializer))
}

func (p *Printer) jsxAttrValue(n *ast.Node) *doc.Doc {
	if n == nil {
		return nil
	}
	if n.Kind == ast.KindStringLiteral {
		p.forceJSXQuote = true
		d := doc.T(p.stringLiteral(n))
		p.forceJSXQuote = false
		return d
	}
	return p.node(n)
}

func (p *Printer) jsxExpr(n *ast.Node) *doc.Doc {
	e := n.AsJsxExpression()
	if e.Expression == nil {
		dangle := p.commentLines(p.takeUntil(n.End()))
		if dangle == nil {
			return doc.T("{}")
		}
		return doc.Concat(doc.T("{"), dangle, doc.T("}"))
	}
	inner := p.node(e.Expression)
	if e.DotDotDotToken != nil {
		inner = doc.Concat(doc.T("..."), inner)
	}
	return doc.Concat(doc.T("{"), inner, p.beforeClose(n.End()), doc.T("}"))
}

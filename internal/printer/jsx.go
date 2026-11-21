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
	children := p.jsxChildren(el.Children)
	close := p.jsxClose(el.ClosingElement)
	if children == nil {
		return doc.Concat(open, close)
	}
	return doc.Group(doc.Concat(
		open,
		doc.Indent(doc.Concat(doc.SoftLine(), children)),
		doc.SoftLine(),
		close,
	))
}

func (p *Printer) jsxFragment(n *ast.Node) *doc.Doc {
	f := n.AsJsxFragment()
	children := p.jsxChildren(f.Children)
	if children == nil {
		return doc.T("<></>")
	}
	return doc.Group(doc.Concat(
		doc.T("<>"),
		doc.Indent(doc.Concat(doc.SoftLine(), children)),
		doc.SoftLine(),
		doc.T("</>"),
	))
}

func (p *Printer) jsxChildren(list *ast.NodeList) *doc.Doc {
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
	if len(parts) == 0 {
		return nil
	}
	return doc.Join(doc.SoftLine(), parts)
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
	if self {
		end = "/>"
	}
	return doc.Group(doc.Concat(
		head,
		doc.Indent(doc.Concat(doc.Line(), attrDoc)),
		doc.SoftLine(),
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
		return doc.T(p.stringLiteral(n))
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

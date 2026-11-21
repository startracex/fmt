// Package doc is a Wadler-style pretty-document IR.
//
// A printer turns a tsgo AST into a document. [Print] then chooses line
// breaks so the result fits a width, which is the same two-step model Prettier
// uses. Groups try to fit on one line; when they cannot, their breaks expand
// and nested groups are measured again.
package doc

import "strings"

// Doc is one node in a pretty-document.
//
// Documents are treated as immutable after they are passed to [Print], except
// that [Print] may mark groups that contain hard breaks so they expand.
type Doc struct {
	kind        kind
	text        string
	parts       []*Doc
	hard        bool
	soft        bool
	shouldBreak bool
	breakDoc    *Doc
	flatDoc     *Doc
}

type kind uint8

const (
	kindEmpty kind = iota
	kindText
	kindConcat
	kindGroup
	kindIndent
	kindLine
	kindIfBreak
)

// T is literal text. Newlines inside text are preserved and do not gain indent,
// which is what template literals need.
func T(s string) *Doc {
	if s == "" {
		return nil
	}
	return &Doc{kind: kindText, text: s}
}

// Concat joins documents in order. Nil entries are skipped.
func Concat(parts ...*Doc) *Doc {
	flat := make([]*Doc, 0, len(parts))
	for _, p := range parts {
		if p == nil {
			continue
		}
		if p.kind == kindConcat {
			flat = append(flat, p.parts...)
			continue
		}
		flat = append(flat, p)
	}
	switch len(flat) {
	case 0:
		return nil
	case 1:
		return flat[0]
	default:
		return &Doc{kind: kindConcat, parts: flat}
	}
}

// Join inserts sep between the non-nil documents.
func Join(sep *Doc, parts []*Doc) *Doc {
	var out []*Doc
	for _, p := range parts {
		if p == nil {
			continue
		}
		if len(out) > 0 && sep != nil {
			out = append(out, sep)
		}
		out = append(out, p)
	}
	return Concat(out...)
}

// Group tries to print d on one line. If it does not fit, breaks inside d expand.
func Group(d *Doc) *Doc {
	if d == nil {
		return nil
	}
	return &Doc{kind: kindGroup, parts: []*Doc{d}}
}

// BreakGroup is a group that always expands.
func BreakGroup(d *Doc) *Doc {
	g := Group(d)
	if g == nil {
		return nil
	}
	g.shouldBreak = true
	return g
}

// ContainsHard reports whether d contains a hard line break.
func ContainsHard(d *Doc) bool {
	if d == nil {
		return false
	}
	if d.kind == kindLine && d.hard {
		return true
	}
	if ContainsHard(d.breakDoc) || ContainsHard(d.flatDoc) {
		return true
	}
	for _, p := range d.parts {
		if ContainsHard(p) {
			return true
		}
	}
	return false
}

// Indent increases the indent by one level for breaks inside d.
func Indent(d *Doc) *Doc {
	if d == nil {
		return nil
	}
	return &Doc{kind: kindIndent, parts: []*Doc{d}}
}

// Line is a space when the enclosing group fits and a newline otherwise.
func Line() *Doc { return &Doc{kind: kindLine} }

// SoftLine is empty when the enclosing group fits and a newline otherwise.
func SoftLine() *Doc { return &Doc{kind: kindLine, soft: true} }

// HardLine always breaks and forces the enclosing group to expand.
func HardLine() *Doc { return &Doc{kind: kindLine, hard: true} }

// IfBreak prints yes when the enclosing group is broken and no when it fits.
func IfBreak(yes, no *Doc) *Doc {
	if yes == nil && no == nil {
		return nil
	}
	return &Doc{kind: kindIfBreak, breakDoc: yes, flatDoc: no}
}

// PrintOptions controls layout.
type PrintOptions struct {
	// Width is the target line length. Zero means 80.
	Width int
	// IndentUnit is one indent level, typically two spaces or a tab.
	IndentUnit string
	// TabWidth is the column width of a tab. Zero means 2.
	TabWidth int
}

// Print lays the document out.
func Print(root *Doc, opt PrintOptions) string {
	if root == nil {
		return ""
	}
	if opt.Width <= 0 {
		opt.Width = 80
	}
	if opt.IndentUnit == "" {
		opt.IndentUnit = "  "
	}
	propagateBreaks(root)
	p := &printer{opt: opt, col: 0}
	p.stack = []cmd{{doc: root}}
	var b strings.Builder
	for len(p.stack) > 0 {
		c := p.pop()
		p.printCmd(&b, c)
	}
	return b.String()
}

// propagateBreaks marks groups that contain a hard break so they expand.
// A hard break inside a nested group does not force the outer group.
func propagateBreaks(d *Doc) bool {
	if d == nil {
		return false
	}
	switch d.kind {
	case kindLine:
		return d.hard
	case kindIfBreak:
		// Only the flat side can force a break. The broken side is printed
		// after the group has already decided to expand.
		return propagateBreaks(d.flatDoc)
	case kindGroup:
		inner := false
		for _, p := range d.parts {
			if propagateBreaks(p) {
				inner = true
			}
		}
		if inner {
			d.shouldBreak = true
		}
		return false
	default:
		force := false
		for _, p := range d.parts {
			if propagateBreaks(p) {
				force = true
			}
		}
		return force
	}
}

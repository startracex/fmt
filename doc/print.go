package doc

import "strings"

type mode uint8

const (
	modeFlat mode = iota
	modeBreak
)

type cmd struct {
	indent int
	mode   mode
	doc    *Doc
}

type printer struct {
	opt   PrintOptions
	stack []cmd
	col   int
}

func (p *printer) pop() cmd {
	c := p.stack[len(p.stack)-1]
	p.stack = p.stack[:len(p.stack)-1]
	return c
}

func (p *printer) push(indent int, m mode, d *Doc) {
	if d == nil {
		return
	}
	p.stack = append(p.stack, cmd{indent: indent, mode: m, doc: d})
}

func (p *printer) printCmd(b *strings.Builder, c cmd) {
	if c.doc == nil {
		return
	}
	d := c.doc
	switch d.kind {
	case kindText:
		p.writeText(b, d.text)
	case kindConcat:
		for i := len(d.parts) - 1; i >= 0; i-- {
			p.push(c.indent, c.mode, d.parts[i])
		}
	case kindIndent:
		// Indent applies only when this group is already breaking. A flat
		// group keeps nested breaks at the current level, so a block on the
		// right of `=` does not pick up a second indent.
		next := c.indent
		if c.mode == modeBreak {
			next++
		}
		p.push(next, c.mode, d.parts[0])
	case kindGroup:
		m := modeFlat
		if d.shouldBreak || !p.groupFits(c.indent, d) {
			m = modeBreak
		}
		for i := len(d.parts) - 1; i >= 0; i-- {
			p.push(c.indent, m, d.parts[i])
		}
	case kindLine:
		p.writeLine(b, c, d)
	case kindIfBreak:
		if c.mode == modeBreak {
			p.push(c.indent, c.mode, d.breakDoc)
		} else {
			p.push(c.indent, c.mode, d.flatDoc)
		}
	default:
	}
}

func (p *printer) writeLine(b *strings.Builder, c cmd, d *Doc) {
	if c.mode == modeFlat && !d.hard {
		if !d.soft {
			b.WriteByte(' ')
			p.col++
		}
		return
	}
	b.WriteByte('\n')
	if p.opt.IndentUnit == "\t" {
		for i := 0; i < c.indent; i++ {
			b.WriteByte('\t')
		}
		p.col = c.indent * p.tabWidth()
	} else {
		unit := p.opt.IndentUnit
		for i := 0; i < c.indent; i++ {
			b.WriteString(unit)
		}
		p.col = c.indent * len(unit)
	}
}

func (p *printer) tabWidth() int {
	if p.opt.TabWidth < 0 {
		return 2
	}
	return p.opt.TabWidth
}

func (p *printer) writeText(b *strings.Builder, s string) {
	for {
		i := strings.IndexByte(s, '\n')
		if i < 0 {
			b.WriteString(s)
			p.col += widthOf(s)
			return
		}
		b.WriteString(s[:i])
		b.WriteByte('\n')
		p.col = 0
		s = s[i+1:]
	}
}

func widthOf(s string) int {
	n := 0
	for _, r := range s {
		if r == '\t' {
			n += 2
			continue
		}
		n++
	}
	return n
}

// groupFits reports whether d's contents fit on the remainder of this line
// when d's own breaks are flat. Nested groups may still break; a newline
// inside them ends the measurement, matching Prettier's fits check.
func (p *printer) groupFits(indent int, d *Doc) bool {
	trial := make([]cmd, len(p.stack), len(p.stack)+len(d.parts))
	copy(trial, p.stack)
	for i := len(d.parts) - 1; i >= 0; i-- {
		trial = append(trial, cmd{indent: indent, mode: modeFlat, doc: d.parts[i]})
	}
	return fits(trial, p.opt.Width-p.col)
}

func fits(stack []cmd, width int) bool {
	s := stack
	for len(s) > 0 {
		if width < 0 {
			return false
		}
		c := s[len(s)-1]
		s = s[:len(s)-1]
		if c.doc == nil {
			continue
		}
		d := c.doc
		switch d.kind {
		case kindText:
			for _, r := range d.text {
				if r == '\n' {
					return true
				}
				if r == '\t' {
					width -= 2
				} else {
					width--
				}
			}
		case kindConcat:
			for i := len(d.parts) - 1; i >= 0; i-- {
				s = append(s, cmd{indent: c.indent, mode: c.mode, doc: d.parts[i]})
			}
		case kindIndent:
			s = append(s, cmd{indent: c.indent + 1, mode: c.mode, doc: d.parts[0]})
		case kindLine:
			if c.mode == modeBreak || d.hard {
				return true
			}
			if !d.soft {
				width--
			}
		case kindIfBreak:
			if c.mode == modeBreak {
				s = append(s, cmd{indent: c.indent, mode: c.mode, doc: d.breakDoc})
			} else {
				s = append(s, cmd{indent: c.indent, mode: c.mode, doc: d.flatDoc})
			}
		case kindGroup:
			m := modeFlat
			if d.shouldBreak {
				m = modeBreak
			} else {
				inner := make([]cmd, len(s), len(s)+len(d.parts))
				copy(inner, s)
				for i := len(d.parts) - 1; i >= 0; i-- {
					inner = append(inner, cmd{indent: c.indent, mode: modeFlat, doc: d.parts[i]})
				}
				if !fits(inner, width) {
					m = modeBreak
				}
			}
			for i := len(d.parts) - 1; i >= 0; i-- {
				s = append(s, cmd{indent: c.indent, mode: m, doc: d.parts[i]})
			}
		}
	}
	return width >= 0
}

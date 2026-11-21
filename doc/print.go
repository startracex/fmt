package doc

import "strings"

type mode uint8

const (
	modeFlat mode = iota
	modeBreak
)

type cmd struct {
	indent int
	extra  int
	mode   mode
	doc    *Doc
}

type printer struct {
	opt           PrintOptions
	stack         []cmd
	col           int
	out           []byte
	atLineStart   bool
	pendingIndent int
	pendingExtra  int
	groupMode     map[int]mode
}

func (p *printer) pop() cmd {
	c := p.stack[len(p.stack)-1]
	p.stack = p.stack[:len(p.stack)-1]
	return c
}

func (p *printer) push(indent, extra int, m mode, d *Doc) {
	if d == nil {
		return
	}
	p.stack = append(p.stack, cmd{indent: indent, extra: extra, mode: m, doc: d})
}

func (p *printer) printCmd(c cmd) {
	if c.doc == nil {
		return
	}
	d := c.doc
	switch d.kind {
	case kindText:
		p.writeText(d.text)
	case kindConcat:
		for i := len(d.parts) - 1; i >= 0; i-- {
			p.push(c.indent, c.extra, c.mode, d.parts[i])
		}
	case kindIndent:
		// A flat group does not indent, so a block that follows `=>` on the
		// same line stays at the current level. Broken groups indent.
		next := c.indent
		if c.mode == modeBreak {
			next++
		}
		p.push(next, c.extra, c.mode, d.parts[0])
	case kindAlign:
		extra := c.extra
		if c.mode == modeBreak {
			extra += d.n
		}
		p.push(c.indent, extra, c.mode, d.parts[0])
	case kindGroup:
		m := modeFlat
		// A flat parent forces its groups flat unless they contain a hard break.
		// Otherwise measure, the same rule Prettier uses from a broken root.
		if c.mode == modeBreak || d.shouldBreak {
			if d.shouldBreak || !p.groupFits(c.indent, c.extra, d) {
				m = modeBreak
			}
		}
		if d.n != 0 {
			if p.groupMode == nil {
				p.groupMode = map[int]mode{}
			}
			p.groupMode[d.n] = m
		}
		for i := len(d.parts) - 1; i >= 0; i-- {
			p.push(c.indent, c.extra, m, d.parts[i])
		}
	case kindLine:
		p.writeLine(c, d)
	case kindIfBreak:
		if c.mode == modeBreak {
			p.push(c.indent, c.extra, c.mode, d.breakDoc)
		} else {
			p.push(c.indent, c.extra, c.mode, d.flatDoc)
		}
	case kindIndentIfBreak:
		p.pushIndentIfBreak(c, d)
	case kindFill:
		p.printFill(c, d)
	default:
	}
}

func (p *printer) writeLine(c cmd, d *Doc) {
	if c.mode == modeFlat && !d.hard {
		if !d.soft {
			p.writeText(" ")
		}
		return
	}
	p.trimTrailingSpace()
	p.out = append(p.out, '\n')
	p.col = 0
	p.atLineStart = true
	p.pendingIndent = c.indent
	p.pendingExtra = c.extra
}

// pending indent is stored in col until content is written, so a blank line
// produced by two breaks in a row does not keep leftover spaces.
func (p *printer) writeIndent() {
	if !p.atLineStart {
		return
	}
	n := p.pendingIndent
	extra := p.pendingExtra
	p.pendingIndent = 0
	p.pendingExtra = 0
	p.atLineStart = false
	if n > 0 {
		if p.opt.IndentUnit == "\t" {
			for i := 0; i < n; i++ {
				p.out = append(p.out, '\t')
			}
			p.col = n * p.tabWidth()
		} else {
			unit := p.opt.IndentUnit
			for i := 0; i < n; i++ {
				p.out = append(p.out, unit...)
			}
			p.col = n * len(unit)
		}
	}
	for i := 0; i < extra; i++ {
		p.out = append(p.out, ' ')
	}
	p.col += extra
}

func (p *printer) trimTrailingSpace() {
	for len(p.out) > 0 {
		last := p.out[len(p.out)-1]
		if last != ' ' && last != '\t' {
			break
		}
		p.out = p.out[:len(p.out)-1]
	}
}

func (p *printer) tabWidth() int {
	if p.opt.TabWidth < 0 {
		return 2
	}
	return p.opt.TabWidth
}

func (p *printer) writeText(s string) {
	if s == "" {
		return
	}
	for {
		i := strings.IndexByte(s, '\n')
		if i < 0 {
			p.writeIndent()
			p.out = append(p.out, s...)
			p.col += widthOf(s)
			return
		}
		if i > 0 {
			p.writeIndent()
			p.out = append(p.out, s[:i]...)
		}
		p.trimTrailingSpace()
		p.out = append(p.out, '\n')
		p.col = 0
		p.atLineStart = true
		p.pendingIndent = 0
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
func (p *printer) groupFits(indent, extra int, d *Doc) bool {
	trial := make([]cmd, len(p.stack), len(p.stack)+len(d.parts))
	copy(trial, p.stack)
	for i := len(d.parts) - 1; i >= 0; i-- {
		trial = append(trial, cmd{indent: indent, extra: extra, mode: modeFlat, doc: d.parts[i]})
	}
	return p.fits(trial, p.opt.Width-p.column())
}

func (p *printer) recordedMode(id int, fallback mode) mode {
	if id == 0 {
		return fallback
	}
	if m, ok := p.groupMode[id]; ok {
		return m
	}
	return modeFlat
}

// printFill lays out alternating content and line documents. A flat parent
// keeps every piece on one line. A broken parent fills the line, then breaks.
func (p *printer) printFill(c cmd, d *Doc) {
	parts := d.parts[d.n:]
	if len(parts) == 0 {
		return
	}
	if c.mode == modeFlat {
		for i := len(parts) - 1; i >= 0; i-- {
			p.push(c.indent, c.extra, modeFlat, parts[i])
		}
		return
	}
	content := parts[0]
	rem := p.opt.Width - p.column()
	contentFits := p.fits([]cmd{{indent: c.indent, extra: c.extra, mode: modeFlat, doc: content}}, rem)
	if len(parts) == 1 {
		mode := modeFlat
		if !contentFits {
			mode = modeBreak
		}
		p.push(c.indent, c.extra, mode, content)
		return
	}
	space := parts[1]
	if len(parts) == 2 {
		if contentFits {
			p.push(c.indent, c.extra, modeFlat, space)
			p.push(c.indent, c.extra, modeFlat, content)
		} else {
			p.push(c.indent, c.extra, modeBreak, space)
			p.push(c.indent, c.extra, modeBreak, content)
		}
		return
	}
	second := parts[2]
	both := p.fits([]cmd{
		{indent: c.indent, extra: c.extra, mode: modeFlat, doc: second},
		{indent: c.indent, extra: c.extra, mode: modeFlat, doc: space},
		{indent: c.indent, extra: c.extra, mode: modeFlat, doc: content},
	}, rem)
	rest := new(Doc)
	*rest = *d
	rest.n += 2
	p.push(c.indent, c.extra, c.mode, rest)
	if both {
		p.push(c.indent, c.extra, modeFlat, space)
		p.push(c.indent, c.extra, modeFlat, content)
	} else if contentFits {
		p.push(c.indent, c.extra, modeBreak, space)
		p.push(c.indent, c.extra, modeFlat, content)
	} else {
		p.push(c.indent, c.extra, modeBreak, space)
		p.push(c.indent, c.extra, modeBreak, content)
	}
}

// pushIndentIfBreak indents d when group n expanded. The mode is recorded
// when that group is printed, which is earlier in the same parent.
func (p *printer) pushIndentIfBreak(c cmd, d *Doc) {
	gm := modeFlat
	if d.n != 0 {
		if m, ok := p.groupMode[d.n]; ok {
			gm = m
		}
	} else {
		gm = c.mode
	}
	indent := c.indent
	if gm == modeBreak && c.mode == modeBreak {
		indent++
	}
	p.push(indent, c.extra, c.mode, d.parts[0])
}

// column is the column where the next text will be written. A break leaves
// the indent pending until text arrives, and that indent still consumes width.
func (p *printer) column() int {
	if !p.atLineStart {
		return p.col
	}
	col := 0
	n := p.pendingIndent
	if n > 0 {
		if p.opt.IndentUnit == "\t" {
			col = n * p.tabWidth()
		} else {
			unit := p.opt.IndentUnit
			if unit == "" {
				col = n * 2
			} else {
				col = n * len(unit)
			}
		}
	}
	return col + p.pendingExtra
}

func (p *printer) fits(stack []cmd, width int) bool {
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
		case kindAlign:
			// Alignment only changes later lines. The first line still counts.
			s = append(s, cmd{indent: c.indent, extra: c.extra, mode: c.mode, doc: d.parts[0]})
		case kindIndentIfBreak:
			ind := c.indent
			if p.recordedMode(d.n, c.mode) == modeBreak && c.mode == modeBreak {
				ind++
			}
			s = append(s, cmd{indent: ind, extra: c.extra, mode: c.mode, doc: d.parts[0]})
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
			// Measure the group as flat unless it must break. Nested groups
			// are not allowed to break on their own during this check, so an
			// outer group only fits when its whole first line fits.
			m := c.mode
			if d.shouldBreak {
				m = modeBreak
			}
			for i := len(d.parts) - 1; i >= 0; i-- {
				s = append(s, cmd{indent: c.indent, mode: m, doc: d.parts[i]})
			}
		}
	}
	return width >= 0
}

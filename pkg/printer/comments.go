package printer

import (
	"strings"

	"github.com/startracex/tsgo/pkg/ast"
	"github.com/startracex/tsgo/pkg/scanner"

	"github.com/startracex/fmt/doc"
)

type comment struct {
	start int
	end   int
	text  string
	block bool
}

// collectComments finds comments in the gaps between leaf tokens. Those gaps
// are trivia plus punctuation, so a comment scan cannot wander into strings,
// templates, or regular expressions.
func collectComments(src string, root *ast.Node) []comment {
	var leaves [][2]int
	collectLeaves(src, root, &leaves)
	if len(leaves) == 0 {
		return scanGap(src, 0, len(src))
	}
	sortSpans(leaves)
	merged := mergeSpans(leaves)
	var out []comment
	pos := 0
	for _, sp := range merged {
		if sp[0] > pos {
			out = append(out, scanGap(src, pos, sp[0])...)
		}
		if sp[1] > pos {
			pos = sp[1]
		}
	}
	if pos < len(src) {
		out = append(out, scanGap(src, pos, len(src))...)
	}
	return out
}

func collectLeaves(src string, n *ast.Node, out *[][2]int) {
	if n == nil || ast.NodeIsMissing(n) || n.Pos() < 0 || n.End() < n.Pos() {
		return
	}
	kids := 0
	n.ForEachChild(func(c *ast.Node) bool {
		if c != nil && !ast.NodeIsMissing(c) {
			kids++
			collectLeaves(src, c, out)
		}
		return false
	})
	if kids != 0 {
		return
	}
	start := n.Pos()
	if start < len(src) {
		start = scanner.SkipTrivia(src, start)
	}
	end := n.End()
	if end > len(src) {
		end = len(src)
	}
	if end > start {
		*out = append(*out, [2]int{start, end})
	}
}

func sortSpans(spans [][2]int) {
	for i := 1; i < len(spans); i++ {
		j := i
		for j > 0 && (spans[j][0] < spans[j-1][0] || (spans[j][0] == spans[j-1][0] && spans[j][1] < spans[j-1][1])) {
			spans[j], spans[j-1] = spans[j-1], spans[j]
			j--
		}
	}
}

func mergeSpans(spans [][2]int) [][2]int {
	out := [][2]int{spans[0]}
	for _, sp := range spans[1:] {
		last := &out[len(out)-1]
		if sp[0] <= last[1] {
			if sp[1] > last[1] {
				last[1] = sp[1]
			}
			continue
		}
		out = append(out, sp)
	}
	return out
}

func scanGap(src string, from, to int) []comment {
	if from < 0 {
		from = 0
	}
	if to > len(src) {
		to = len(src)
	}
	var out []comment
	i := from
	for i < to {
		if src[i] == '/' && i+1 < to && (src[i+1] == '/' || src[i+1] == '*') {
			if src[i+1] == '/' {
				j := i + 2
				for j < to && src[j] != '\n' && src[j] != '\r' {
					j++
				}
				out = append(out, comment{start: i, end: j, text: src[i:j]})
				i = j
				continue
			}
			j := i + 2
			for j+1 < to && !(src[j] == '*' && src[j+1] == '/') {
				j++
			}
			if j+1 < to {
				j += 2
			} else {
				j = to
			}
			out = append(out, comment{start: i, end: j, text: src[i:j], block: true})
			i = j
			continue
		}
		i++
	}
	return out
}

func (p *Printer) takeUntil(pos int) []comment {
	var out []comment
	for p.ci < len(p.comments) && p.comments[p.ci].start < pos {
		out = append(out, p.comments[p.ci])
		p.ci++
	}
	return out
}

func (p *Printer) discardUntil(pos int) {
	for p.ci < len(p.comments) && p.comments[p.ci].start < pos {
		p.ci++
	}
}

func (p *Printer) leading(cs []comment, nodeStart int) *doc.Doc {
	if len(cs) == 0 {
		return nil
	}
	var parts []*doc.Doc
	for i, c := range cs {
		if i > 0 {
			parts = append(parts, gapBreak(p.src, cs[i-1].end, c.start))
		}
		parts = append(parts, doc.T(c.text))
	}
	last := cs[len(cs)-1]
	if !last.block || hasNewline(p.src, last.end, nodeStart) {
		parts = append(parts, doc.HardLine())
	} else {
		parts = append(parts, doc.T(" "))
	}
	return doc.Concat(parts...)
}

func (p *Printer) inlineComments(cs []comment) *doc.Doc {
	if len(cs) == 0 {
		return nil
	}
	var parts []*doc.Doc
	for i, c := range cs {
		if i == 0 {
			parts = append(parts, doc.T(" "))
		} else {
			parts = append(parts, gapBreak(p.src, cs[i-1].end, c.start))
		}
		parts = append(parts, doc.T(c.text))
	}
	return doc.Concat(parts...)
}

func (p *Printer) commentLines(cs []comment) *doc.Doc {
	if len(cs) == 0 {
		return nil
	}
	parts := make([]*doc.Doc, 0, len(cs)*2)
	for i, c := range cs {
		if i > 0 {
			parts = append(parts, gapBreak(p.src, cs[i-1].end, c.start))
			if !hasNewline(p.src, cs[i-1].end, c.start) {
				parts = append(parts, doc.HardLine())
			}
		}
		parts = append(parts, doc.T(c.text))
	}
	return doc.Concat(parts...)
}

func gapBreak(src string, from, to int) *doc.Doc {
	if blankBetween(src, from, to) {
		return doc.Concat(doc.HardLine(), doc.HardLine())
	}
	if hasNewline(src, from, to) {
		return doc.HardLine()
	}
	return doc.T(" ")
}

func (p *Printer) takeSameLine(prevEnd, limit int) *doc.Doc {
	if prevEnd < 0 {
		return nil
	}
	var parts []*doc.Doc
	for p.ci < len(p.comments) && p.comments[p.ci].start < limit {
		c := p.comments[p.ci]
		gapFrom := prevEnd
		if gapFrom > c.start {
			gapFrom = c.start
		}
		if hasNewline(p.src, gapFrom, c.start) {
			break
		}
		parts = append(parts, doc.T(" "), doc.T(c.text))
		prevEnd = c.end
		p.ci++
	}
	return doc.Concat(parts...)
}

func hasNewline(src string, from, to int) bool {
	if from < 0 {
		from = 0
	}
	if to > len(src) {
		to = len(src)
	}
	if to <= from {
		return false
	}
	return strings.Contains(src[from:to], "\n") || strings.Contains(src[from:to], "\r")
}

func blankBetween(src string, from, to int) bool {
	if from < 0 {
		from = 0
	}
	if to > len(src) {
		to = len(src)
	}
	if to <= from {
		return false
	}
	// A blank line is a newline followed only by spaces and another newline.
	// A comment on its own line is not a blank line.
	s := src[from:to]
	for i := 0; i < len(s); i++ {
		if s[i] != '\n' {
			continue
		}
		j := i + 1
		for j < len(s) && (s[j] == ' ' || s[j] == '\t' || s[j] == '\r') {
			j++
		}
		if j < len(s) && s[j] == '\n' {
			return true
		}
	}
	return false
}

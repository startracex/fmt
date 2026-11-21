// Package ignore matches paths against a subset of gitignore rules.
package ignore

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Set is an ordered list of gitignore-style patterns. Later rules win.
type Set struct {
	patterns []pattern
}

type pattern struct {
	negate  bool
	dirOnly bool
	re      *regexp.Regexp
}

// Parse reads gitignore-style lines.
func Parse(text string) (*Set, error) {
	s := &Set{}
	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p, ok, err := compile(line)
		if err != nil {
			return nil, err
		}
		if ok {
			s.patterns = append(s.patterns, p)
		}
	}
	return s, sc.Err()
}

// Load reads a pattern file. A missing file yields an empty set.
func Load(path string) (*Set, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Set{}, nil
		}
		return nil, err
	}
	return Parse(string(data))
}

// Match reports whether a slash-separated relative path is ignored.
// isDir must be true for directories so that directory-only rules apply.
func (s *Set) Match(rel string, isDir bool) bool {
	if s == nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	rel = strings.TrimPrefix(rel, "./")
	rel = strings.Trim(rel, "/")
	matched := false
	for _, p := range s.patterns {
		if p.dirOnly && !isDir {
			continue
		}
		if p.re.MatchString(rel) {
			matched = !p.negate
		}
	}
	return matched
}

func compile(line string) (pattern, bool, error) {
	neg := false
	if strings.HasPrefix(line, "!") {
		neg = true
		line = strings.TrimSpace(line[1:])
	}
	if line == "" {
		return pattern{}, false, nil
	}
	dirOnly := strings.HasSuffix(line, "/")
	line = strings.TrimSuffix(line, "/")
	anchored := strings.HasPrefix(line, "/") || strings.Contains(strings.TrimPrefix(line, "/"), "/")
	line = strings.TrimPrefix(line, "/")
	expr := glob(line)
	if anchored {
		expr = "^" + expr + "$"
	} else {
		expr = "(^|/)" + expr + "$"
	}
	re, err := regexp.Compile(expr)
	if err != nil {
		return pattern{}, false, err
	}
	return pattern{negate: neg, dirOnly: dirOnly, re: re}, true, nil
}

func glob(pat string) string {
	var b strings.Builder
	for i := 0; i < len(pat); i++ {
		switch pat[i] {
		case '*':
			if i+1 < len(pat) && pat[i+1] == '*' {
				i++
				if i+1 < len(pat) && pat[i+1] == '/' {
					i++
					b.WriteString("(?:.*/)?")
					continue
				}
				b.WriteString(".*")
				continue
			}
			b.WriteString("[^/]*")
		case '?':
			b.WriteString("[^/]")
		case '.', '+', '(', ')', '|', '^', '$', '{', '}', '[', ']', '\\':
			b.WriteByte('\\')
			b.WriteByte(pat[i])
		default:
			b.WriteByte(pat[i])
		}
	}
	return b.String()
}

// DefaultNames are directory names that are always skipped.
var DefaultNames = []string{"node_modules", ".git", "dist", "build", "coverage", "vendor"}

// Always reports whether rel has a path component that formatters always skip.
func Always(rel string) bool {
	rel = filepath.ToSlash(rel)
	for _, part := range strings.Split(rel, "/") {
		for _, name := range DefaultNames {
			if part == name {
				return true
			}
		}
	}
	return false
}

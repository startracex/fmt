// Package editorconfig reads .editorconfig files and maps them onto formatter options.
package editorconfig

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/startracex/fmt/internal/opt"
)

// Load returns the EditorConfig properties that apply to filename, mapped onto
// a partial option set. A missing file is not an error.
func Load(filename string) (opt.Partial, error) {
	if filename == "" {
		return opt.Partial{}, nil
	}
	abs, err := filepath.Abs(filename)
	if err != nil {
		return opt.Partial{}, err
	}
	configs, err := discover(filepath.Dir(abs))
	if err != nil || len(configs) == 0 {
		return opt.Partial{}, err
	}
	props := map[string]string{}
	// Parent files first so a closer file overrides them.
	for i := len(configs) - 1; i >= 0; i-- {
		cfg := configs[i]
		rel, err := filepath.Rel(filepath.Dir(cfg.path), abs)
		if err != nil {
			return opt.Partial{}, err
		}
		rel = filepath.ToSlash(rel)
		for _, sec := range cfg.sections {
			if !sec.matches(rel) {
				continue
			}
			for k, v := range sec.props {
				props[k] = v
			}
		}
	}
	return fromProps(props), nil
}

type parsed struct {
	path     string
	root     bool
	sections []section
}

type section struct {
	pattern string
	props   map[string]string
}

func (s section) matches(rel string) bool {
	re, err := glob(s.pattern)
	if err != nil {
		return false
	}
	return re.MatchString(rel)
}

func discover(dir string) ([]parsed, error) {
	var out []parsed
	for {
		candidate := filepath.Join(dir, ".editorconfig")
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
			cfg, err := parseFile(candidate)
			if err != nil {
				return nil, err
			}
			out = append(out, cfg)
			if cfg.root {
				break
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return out, nil
}

func parseFile(path string) (parsed, error) {
	f, err := os.Open(path)
	if err != nil {
		return parsed{}, err
	}
	defer f.Close()
	cfg := parsed{path: path}
	var cur *section
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(sc.Text(), "\uFEFF"))
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			cfg.sections = append(cfg.sections, section{
				pattern: line[1 : len(line)-1],
				props:   map[string]string{},
			})
			cur = &cfg.sections[len(cfg.sections)-1]
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		val = strings.TrimSpace(val)
		if hash := strings.Index(val, " #"); hash >= 0 {
			val = strings.TrimSpace(val[:hash])
		}
		if semi := strings.Index(val, " ;"); semi >= 0 {
			val = strings.TrimSpace(val[:semi])
		}
		val = strings.ToLower(val)
		if cur == nil {
			if key == "root" && (val == "true" || val == "false") {
				cfg.root = val == "true"
			}
			continue
		}
		cur.props[key] = val
	}
	return cfg, sc.Err()
}

func fromProps(props map[string]string) opt.Partial {
	var out opt.Partial
	if style, ok := props["indent_style"]; ok {
		switch style {
		case "tab":
			out.UseTabs = boolPtr(true)
		case "space":
			out.UseTabs = boolPtr(false)
		}
	}
	if size, ok := props["indent_size"]; ok {
		if size == "tab" {
			out.UseTabs = boolPtr(true)
		} else if n, ok := atoi(size); ok {
			out.TabWidth = intPtr(n)
		}
	}
	if width, ok := props["tab_width"]; ok {
		if n, ok := atoi(width); ok {
			out.TabWidth = intPtr(n)
		}
	}
	if n, ok := props["max_line_length"]; ok && n != "off" {
		if v, ok := atoi(n); ok {
			out.PrintWidth = intPtr(v)
		}
	}
	if eol, ok := props["end_of_line"]; ok {
		switch eol {
		case "lf", "crlf", "cr":
			out.EndOfLine = strPtr(eol)
		}
	}
	if q, ok := props["quote_type"]; ok {
		switch q {
		case "single":
			out.SingleQuote = boolPtr(true)
		case "double":
			out.SingleQuote = boolPtr(false)
		}
	}
	return out
}

func atoi(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	n, err := strconv.Atoi(s)
	return n, err == nil
}

func glob(pattern string) (*regexp.Regexp, error) {
	anchored := strings.Contains(pattern, "/")
	var b strings.Builder
	b.WriteString("(?i)")
	if !anchored {
		b.WriteString("(^|/)")
	} else {
		b.WriteByte('^')
	}
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				i++
				if i+1 < len(pattern) && pattern[i+1] == '/' {
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
		case '{':
			end := strings.IndexByte(pattern[i:], '}')
			if end < 0 {
				b.WriteString(`\{`)
				continue
			}
			body := pattern[i+1 : i+end]
			i += end
			parts := strings.Split(body, ",")
			b.WriteString("(?:")
			for j, part := range parts {
				if j > 0 {
					b.WriteByte('|')
				}
				b.WriteString(regexp.QuoteMeta(part))
			}
			b.WriteByte(')')
		case '\\':
			if i+1 < len(pattern) {
				i++
				b.WriteString(regexp.QuoteMeta(pattern[i : i+1]))
			}
		default:
			if strings.ContainsRune(`.+()|[]^$`, rune(pattern[i])) || unicode.IsSpace(rune(pattern[i])) {
				b.WriteString(regexp.QuoteMeta(pattern[i : i+1]))
			} else {
				b.WriteByte(pattern[i])
			}
		}
	}
	b.WriteByte('$')
	return regexp.Compile(b.String())
}

func intPtr(n int) *int       { return &n }
func boolPtr(v bool) *bool    { return &v }
func strPtr(s string) *string { return &s }

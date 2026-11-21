// Package editorconfig reads .editorconfig files and maps them onto formatter options.
package editorconfig

import (
	"strconv"
	"strings"

	editorconfig "github.com/editorconfig/editorconfig-core-go/v2"

	"github.com/startracex/fmt/pkg/opt"
)

// Load returns the EditorConfig properties that apply to filename, mapped onto
// a partial option set. A missing file is not an error. Parsing and glob
// matching come from editorconfig-core-go.
func Load(filename string) (opt.Partial, error) {
	if filename == "" {
		return opt.Partial{}, nil
	}
	def, err := editorconfig.GetDefinitionForFilename(filename)
	if err != nil {
		return opt.Partial{}, err
	}
	if def == nil {
		return opt.Partial{}, nil
	}
	return fromDefinition(def), nil
}

func fromDefinition(def *editorconfig.Definition) opt.Partial {
	var out opt.Partial
	raw := def.Raw
	if raw == nil {
		raw = map[string]string{}
	}
	switch strings.ToLower(def.IndentStyle) {
	case editorconfig.IndentStyleTab:
		out.UseTabs = boolPtr(true)
	case editorconfig.IndentStyleSpaces:
		out.UseTabs = boolPtr(false)
	}
	if size, ok := raw["indent_size"]; ok && !unset(size) {
		if strings.EqualFold(size, "tab") {
			out.UseTabs = boolPtr(true)
		} else if n, ok := atoi(size); ok {
			out.TabWidth = intPtr(n)
		}
	}
	// tab_width is its own property and wins over indent_size when both match.
	if width, ok := raw["tab_width"]; ok && !unset(width) {
		if n, ok := atoi(width); ok {
			out.TabWidth = intPtr(n)
		}
	}
	if n, ok := raw["max_line_length"]; ok && !unset(n) && !strings.EqualFold(n, "off") {
		if v, ok := atoi(n); ok {
			out.PrintWidth = intPtr(v)
		}
	}
	eol := def.EndOfLine
	if eol == "" {
		eol = raw["end_of_line"]
	}
	switch strings.ToLower(eol) {
	case "lf", "crlf", "cr":
		out.EndOfLine = strPtr(strings.ToLower(eol))
	}
	if q, ok := raw["quote_type"]; ok && !unset(q) {
		switch strings.ToLower(q) {
		case "single":
			out.SingleQuote = boolPtr(true)
		case "double":
			out.SingleQuote = boolPtr(false)
		}
	}
	return out
}

func unset(s string) bool {
	return strings.EqualFold(strings.TrimSpace(s), "unset")
}

func atoi(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	return n, err == nil && n >= 0
}

func intPtr(n int) *int       { return &n }
func boolPtr(v bool) *bool    { return &v }
func strPtr(s string) *string { return &s }

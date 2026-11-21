// Package config loads formatter options from a JSON config file.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/startracex/fmt/internal/opt"
)

// names are searched from a file's directory upward.
var names = []string{".fmtrc.json", "fmt.config.json", ".fmtrc"}

// Load merges the nearest config into base. A missing file leaves base unchanged.
func Load(startDir string, base opt.Options) (opt.Options, error) {
	path, err := Find(startDir)
	if err != nil || path == "" {
		return base, err
	}
	return LoadFile(path, base)
}

// Find returns the nearest config path, or "" if there is none.
func Find(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	for {
		for _, name := range names {
			candidate := filepath.Join(dir, name)
			if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
				return candidate, nil
			}
		}
		pkg := filepath.Join(dir, "package.json")
		if st, err := os.Stat(pkg); err == nil && !st.IsDir() {
			if ok, err := packageHasFmt(pkg); err != nil {
				return "", err
			} else if ok {
				return pkg, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}

// LoadFile applies a config file onto base.
func LoadFile(path string, base opt.Options) (opt.Options, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return base, err
	}
	if filepath.Base(path) == "package.json" {
		var wrap struct {
			Fmt json.RawMessage `json:"fmt"`
		}
		if err := json.Unmarshal(data, &wrap); err != nil {
			return base, err
		}
		data = wrap.Fmt
	}
	return apply(data, base)
}

func packageHasFmt(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	var wrap struct {
		Fmt json.RawMessage `json:"fmt"`
	}
	if err := json.Unmarshal(data, &wrap); err != nil {
		return false, err
	}
	return len(wrap.Fmt) > 0 && string(wrap.Fmt) != "null", nil
}

type file struct {
	PrintWidth     *int    `json:"printWidth"`
	TabWidth       *int    `json:"tabWidth"`
	UseTabs        *bool   `json:"useTabs"`
	Semi           *bool   `json:"semi"`
	SingleQuote    *bool   `json:"singleQuote"`
	TrailingComma  *string `json:"trailingComma"`
	BracketSpacing *bool   `json:"bracketSpacing"`
	ArrowParens    *string `json:"arrowParens"`
	EndOfLine      *string `json:"endOfLine"`
	QuoteProps     *string `json:"quoteProps"`
}

func apply(data []byte, base opt.Options) (opt.Options, error) {
	var f file
	if err := json.Unmarshal(data, &f); err != nil {
		return base, err
	}
	if f.PrintWidth != nil {
		base.PrintWidth = *f.PrintWidth
	}
	if f.TabWidth != nil {
		base.TabWidth = *f.TabWidth
	}
	if f.UseTabs != nil {
		base.UseTabs = *f.UseTabs
	}
	if f.Semi != nil {
		base.Semi = *f.Semi
	}
	if f.SingleQuote != nil {
		base.SingleQuote = *f.SingleQuote
	}
	if f.TrailingComma != nil {
		base.TrailingComma = strings.ToLower(*f.TrailingComma)
	}
	if f.BracketSpacing != nil {
		base.BracketSpacing = *f.BracketSpacing
	}
	if f.ArrowParens != nil {
		base.ArrowParens = strings.ToLower(*f.ArrowParens)
	}
	if f.EndOfLine != nil {
		base.EndOfLine = strings.ToLower(*f.EndOfLine)
	}
	if f.QuoteProps != nil {
		base.QuoteProps = strings.ToLower(*f.QuoteProps)
	}
	return base.Normalized(), nil
}

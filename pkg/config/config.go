// Package config loads formatter options from a JSON config file.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/startracex/fmt/pkg/opt"
)

// names are searched from a file's directory upward.
var names = []string{".fmtrc.json", "fmt.config.json", ".fmtrc"}

// Load reads the nearest config. A missing file returns an empty partial.
func Load(startDir string) (opt.Partial, error) {
	path, err := Find(startDir)
	if err != nil || path == "" {
		return opt.Partial{}, err
	}
	return LoadFile(path)
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

// LoadFile reads one config file. Unset fields stay nil, including when the
// file explicitly sets a zero or false.
func LoadFile(path string) (opt.Partial, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return opt.Partial{}, err
	}
	if filepath.Base(path) == "package.json" {
		var wrap struct {
			Fmt json.RawMessage `json:"fmt"`
		}
		if err := json.Unmarshal(data, &wrap); err != nil {
			return opt.Partial{}, err
		}
		data = wrap.Fmt
	}
	return decode(data)
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

func decode(data []byte) (opt.Partial, error) {
	var partial opt.Partial
	if len(data) == 0 || string(data) == "null" {
		return partial, nil
	}
	if err := json.Unmarshal(data, &partial); err != nil {
		return opt.Partial{}, err
	}
	lower := func(s *string) *string {
		if s == nil {
			return nil
		}
		v := strings.ToLower(*s)
		return &v
	}
	partial.TrailingComma = lower(partial.TrailingComma)
	partial.ArrowParens = lower(partial.ArrowParens)
	partial.EndOfLine = lower(partial.EndOfLine)
	partial.QuoteProps = lower(partial.QuoteProps)
	partial.ObjectWrap = lower(partial.ObjectWrap)
	return partial, nil
}

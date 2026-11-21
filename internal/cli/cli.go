// Package cli implements the fmt command.
package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/startracex/fmt/format"
	"github.com/startracex/fmt/internal/config"
	"github.com/startracex/fmt/internal/ignore"
	"github.com/startracex/fmt/internal/opt"
)

// Main runs the formatter and returns a process status.
func Main(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fmt", flag.ContinueOnError)
	fs.SetOutput(stderr)
	check := fs.Bool("check", false, "do not write; exit 1 if a file would change")
	write := fs.Bool("write", true, "write formatted files in place")
	printWidth := fs.Int("print-width", 0, "line width (default 80)")
	tabWidth := fs.Int("tab-width", 0, "spaces per indent (default 2)")
	useTabs := fs.Bool("use-tabs", false, "indent with tabs")
	semi := fs.Bool("semi", true, "print semicolons")
	single := fs.Bool("single-quote", false, "prefer single quotes")
	trailing := fs.String("trailing-comma", "", "all, es5, or none")
	bracket := fs.Bool("bracket-spacing", true, "spaces inside object braces")
	arrow := fs.String("arrow-parens", "", "always or avoid")
	eol := fs.String("end-of-line", "", "lf, crlf, or cr")
	quotes := fs.String("quote-props", "", "as-needed, consistent, or preserve")
	configPath := fs.String("config", "", "config file (default: nearest .fmtrc.json)")
	ignorePath := fs.String("ignore-path", "", "ignore file (default: nearest .fmtignore)")
	stdinPath := fs.String("stdin-filepath", "stdin.ts", "path used for stdin language and config lookup")
	fs.Usage = func() {
		fmt.Fprintf(stderr, "usage: fmt [flags] [path...]\n\n")
		fmt.Fprintf(stderr, "Formats TypeScript and JavaScript. Paths may be files or directories.\n")
		fmt.Fprintf(stderr, "With no paths, reads stdin and writes stdout.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	opts := format.DefaultOptions()
	paths := fs.Args()
	if len(paths) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(stderr, "stdin: %v\n", err)
			return 2
		}
		opts = resolve(filepath.Dir(*stdinPath), *configPath, opts, stderr)
		opts = applyFlags(opts, fs, *printWidth, *tabWidth, *useTabs, *semi, *single, *trailing, *bracket, *arrow, *eol, *quotes)
		out, err := format.Source(*stdinPath, string(data), opts)
		if err != nil {
			fmt.Fprintf(stderr, "%v\n", err)
			return 2
		}
		if *check && out != normalizeEOL(string(data), opts.EndOfLine) && out != string(data) {
			fmt.Fprintln(stdout, *stdinPath)
			return 1
		}
		if !*check {
			fmt.Fprint(stdout, out)
		}
		return 0
	}

	ig, err := loadIgnore(*ignorePath, paths)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 2
	}
	var files []string
	for _, path := range paths {
		found, err := collect(path, ig)
		if err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", path, err)
			return 2
		}
		files = append(files, found...)
	}
	if len(files) == 0 {
		fmt.Fprintln(stderr, "no files to format")
		return 0
	}
	changed := 0
	failed := 0
	for _, file := range files {
		opts = resolve(filepath.Dir(file), *configPath, format.DefaultOptions(), stderr)
		opts = applyFlags(opts, fs, *printWidth, *tabWidth, *useTabs, *semi, *single, *trailing, *bracket, *arrow, *eol, *quotes)
		src, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", file, err)
			failed++
			continue
		}
		out, err := format.Source(file, string(src), opts)
		if err != nil {
			fmt.Fprintf(stderr, "%v\n", err)
			failed++
			continue
		}
		if out == string(src) {
			continue
		}
		changed++
		if *check || !*write {
			fmt.Fprintln(stdout, file)
			continue
		}
		if err := os.WriteFile(file, []byte(out), 0o644); err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", file, err)
			failed++
			continue
		}
		fmt.Fprintln(stdout, file)
	}
	if failed > 0 {
		return 2
	}
	if *check && changed > 0 {
		return 1
	}
	return 0
}

func resolve(dir, explicit string, base opt.Options, stderr io.Writer) opt.Options {
	var (
		opts opt.Options
		err  error
	)
	if explicit != "" {
		opts, err = config.LoadFile(explicit, base)
	} else {
		opts, err = config.Load(dir, base)
	}
	if err != nil {
		fmt.Fprintf(stderr, "config: %v\n", err)
		return base
	}
	return opts
}

func applyFlags(base opt.Options, fs *flag.FlagSet, width, tab int, tabs, semi, single bool, trailing string, bracket bool, arrow, eol, quotes string) opt.Options {
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	if set["print-width"] {
		base.PrintWidth = width
	}
	if set["tab-width"] {
		base.TabWidth = tab
	}
	if set["use-tabs"] {
		base.UseTabs = tabs
	}
	if set["semi"] {
		base.Semi = semi
	}
	if set["single-quote"] {
		base.SingleQuote = single
	}
	if set["trailing-comma"] {
		base.TrailingComma = strings.ToLower(trailing)
	}
	if set["bracket-spacing"] {
		base.BracketSpacing = bracket
	}
	if set["arrow-parens"] {
		base.ArrowParens = strings.ToLower(arrow)
	}
	if set["end-of-line"] {
		base.EndOfLine = strings.ToLower(eol)
	}
	if set["quote-props"] {
		base.QuoteProps = strings.ToLower(quotes)
	}
	return base.Normalized()
}

func loadIgnore(explicit string, paths []string) (*ignore.Set, error) {
	if explicit != "" {
		return ignore.Load(explicit)
	}
	start, err := os.Getwd()
	if err != nil {
		return &ignore.Set{}, nil
	}
	if len(paths) == 1 {
		if st, err := os.Stat(paths[0]); err == nil && st.IsDir() {
			start = paths[0]
		}
	}
	dir, err := filepath.Abs(start)
	if err != nil {
		return &ignore.Set{}, nil
	}
	for {
		candidate := filepath.Join(dir, ".fmtignore")
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
			return ignore.Load(candidate)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return &ignore.Set{}, nil
		}
		dir = parent
	}
}

func collect(path string, ig *ignore.Set) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		if !wantedExt(path) {
			return nil, nil
		}
		return []string{path}, nil
	}
	root, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	var files []string
	err = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if ignore.Always(rel) || (ig != nil && ig.Match(rel, d.IsDir())) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if wantedExt(p) {
			files = append(files, p)
		}
		return nil
	})
	return files, err
}

func wantedExt(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".ts", ".tsx", ".js", ".jsx", ".mts", ".cts", ".mjs", ".cjs":
		return true
	default:
		return false
	}
}

func normalizeEOL(s, eol string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	switch eol {
	case "crlf":
		return strings.ReplaceAll(s, "\n", "\r\n")
	case "cr":
		return strings.ReplaceAll(s, "\n", "\r")
	default:
		return s
	}
}

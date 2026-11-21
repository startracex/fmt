// Package format pretty-prints TypeScript and JavaScript.
//
// The printer walks a tsgo syntax tree into a width-aware document and reprints
// it. It does not use tsgo's language-service formatter.
package format

import (
	"strings"

	"github.com/startracex/tsgo/pkg/ast"
	"github.com/startracex/tsgo/pkg/core"
	"github.com/startracex/tsgo/pkg/parser"
	"github.com/startracex/tsgo/pkg/tspath"

	"github.com/startracex/fmt/internal/opt"
	"github.com/startracex/fmt/internal/printer"
)

// Options controls layout. See [DefaultOptions].
type Options = opt.Options

// DefaultOptions returns Prettier-like defaults.
func DefaultOptions() Options { return opt.Default() }

// Source formats source text. filename is used to pick the script kind
// (.ts, .tsx, .js, .jsx, .mts, .cts, .mjs, .cjs).
func Source(filename, src string, options Options) (string, error) {
	file, err := Parse(filename, src)
	if err != nil {
		return "", err
	}
	return Print(file, options), nil
}

// Parse parses src with tsgo. Syntax errors are returned.
func Parse(filename, src string) (*ast.SourceFile, error) {
	path := rootedFileName(filename)
	kind := core.EnsureScriptKindFromFileName(path)
	file := parser.ParseSourceFile(ast.SourceFileParseOptions{
		FileName: path,
		PathKey:  tspath.CaseSensitive.PathKey(tspath.RootedPath(path)),
	}, src, kind)
	if diags := file.Diagnostics(); len(diags) > 0 {
		d := diags[0]
		return nil, &SyntaxError{File: filename, Pos: d.Pos(), Message: d.String()}
	}
	return file, nil
}

// Print formats an already-parsed file.
func Print(file *ast.SourceFile, options Options) string {
	if file == nil {
		return ""
	}
	options = options.Normalized()
	out := printer.PrintFile(file, options)
	if strings.TrimSpace(out) == "" {
		return ""
	}
	out = strings.TrimRight(out, " \t")
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	switch options.EndOfLine {
	case "crlf":
		out = strings.ReplaceAll(out, "\n", "\r\n")
	case "cr":
		out = strings.ReplaceAll(out, "\n", "\r")
	}
	return out
}

// SyntaxError is a parse failure.
type SyntaxError struct {
	File    string
	Pos     int
	Message string
}

func (e *SyntaxError) Error() string {
	if e.File == "" {
		return e.Message
	}
	return e.File + ": " + e.Message
}

// rootedFileName turns a user path into the rooted, slash-normalized form tsgo
// requires. Absolute DOS, UNC, URL, and POSIX paths are normalized in place.
// Relative paths, including Windows `src\a.ts`, are anchored at `/` so the
// extension is preserved without pretending a backslash path is POSIX.
func rootedFileName(name string) tspath.RootedFilePath {
	if name == "" {
		name = "file.ts"
	}
	if path, ok := tspath.TryRootedFilePathFromAbsolute(name); ok {
		return path
	}
	return tspath.ToRootedFilePath(name, tspath.RootedDirectoryPathFromNormalized("/"))
}

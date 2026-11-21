// Package opt holds formatter options shared by the library and the CLI.
package opt

// Options controls pretty-printing. Zero values are not the defaults;
// use [Default].
type Options struct {
	PrintWidth     int
	TabWidth       int
	UseTabs        bool
	Semi           bool
	SingleQuote    bool
	TrailingComma  string // all, es5, none
	BracketSpacing bool
	ArrowParens    string // always, avoid
	EndOfLine      string // lf, crlf, cr
	QuoteProps     string // as-needed, consistent, preserve
}

// Default returns Prettier-like defaults: width 80, two spaces, semicolons,
// double quotes, and trailing commas wherever the language allows them.
func Default() Options {
	return Options{
		PrintWidth:     80,
		TabWidth:       2,
		Semi:           true,
		TrailingComma:  "all",
		BracketSpacing: true,
		ArrowParens:    "always",
		EndOfLine:      "lf",
		QuoteProps:     "as-needed",
	}
}

// Normalized fills empty fields with defaults and clamps illegal values.
func (o Options) Normalized() Options {
	d := Default()
	if o.PrintWidth <= 0 {
		o.PrintWidth = d.PrintWidth
	}
	if o.TabWidth <= 0 {
		o.TabWidth = d.TabWidth
	}
	switch o.TrailingComma {
	case "all", "es5", "none":
	default:
		o.TrailingComma = d.TrailingComma
	}
	switch o.ArrowParens {
	case "always", "avoid":
	default:
		o.ArrowParens = d.ArrowParens
	}
	switch o.EndOfLine {
	case "lf", "crlf", "cr":
	default:
		o.EndOfLine = d.EndOfLine
	}
	switch o.QuoteProps {
	case "as-needed", "consistent", "preserve":
	default:
		o.QuoteProps = d.QuoteProps
	}
	return o
}

// TrailKind is a syntactic position that may take a trailing comma.
type TrailKind int

const (
	TrailNone TrailKind = iota
	TrailArray
	TrailObject
	TrailImport
	TrailArgs
	TrailParams
	TrailType
)

// Trailing reports whether a broken list in this position gets a trailing comma.
func (o Options) Trailing(kind TrailKind) bool {
	switch o.TrailingComma {
	case "none":
		return false
	case "es5":
		return kind == TrailArray || kind == TrailObject || kind == TrailImport
	default:
		return kind != TrailNone
	}
}

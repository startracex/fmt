// Package opt holds formatter options shared by the library and the CLI.
package opt

import "strings"

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
	// ObjectWrap is preserve or collapse. preserve keeps a collection expanded
	// when the source has a line break after { or [.
	ObjectWrap string
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
		ObjectWrap:     "preserve",
	}
}

// Partial is a config layer. A nil field is unset and must not override a
// value from a lower layer, including explicit zeros and false.
type Partial struct {
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
	ObjectWrap     *string `json:"objectWrap"`
}

// Merge returns a copy of p with every non-nil field of over applied on top.
func (p Partial) Merge(over Partial) Partial {
	if over.PrintWidth != nil {
		p.PrintWidth = over.PrintWidth
	}
	if over.TabWidth != nil {
		p.TabWidth = over.TabWidth
	}
	if over.UseTabs != nil {
		p.UseTabs = over.UseTabs
	}
	if over.Semi != nil {
		p.Semi = over.Semi
	}
	if over.SingleQuote != nil {
		p.SingleQuote = over.SingleQuote
	}
	if over.TrailingComma != nil {
		p.TrailingComma = over.TrailingComma
	}
	if over.BracketSpacing != nil {
		p.BracketSpacing = over.BracketSpacing
	}
	if over.ArrowParens != nil {
		p.ArrowParens = over.ArrowParens
	}
	if over.EndOfLine != nil {
		p.EndOfLine = over.EndOfLine
	}
	if over.QuoteProps != nil {
		p.QuoteProps = over.QuoteProps
	}
	if over.ObjectWrap != nil {
		p.ObjectWrap = over.ObjectWrap
	}
	return p
}

// Resolve fills nil fields from [Default]. Explicit zeros and false stay.
func (p Partial) Resolve() Options {
	o := Default()
	if p.PrintWidth != nil {
		o.PrintWidth = *p.PrintWidth
	}
	if p.TabWidth != nil {
		o.TabWidth = *p.TabWidth
	}
	if p.UseTabs != nil {
		o.UseTabs = *p.UseTabs
	}
	if p.Semi != nil {
		o.Semi = *p.Semi
	}
	if p.SingleQuote != nil {
		o.SingleQuote = *p.SingleQuote
	}
	if p.TrailingComma != nil {
		o.TrailingComma = strings.ToLower(*p.TrailingComma)
	}
	if p.BracketSpacing != nil {
		o.BracketSpacing = *p.BracketSpacing
	}
	if p.ArrowParens != nil {
		o.ArrowParens = strings.ToLower(*p.ArrowParens)
	}
	if p.EndOfLine != nil {
		o.EndOfLine = strings.ToLower(*p.EndOfLine)
	}
	if p.QuoteProps != nil {
		o.QuoteProps = strings.ToLower(*p.QuoteProps)
	}
	if p.ObjectWrap != nil {
		o.ObjectWrap = strings.ToLower(*p.ObjectWrap)
	}
	return o.Normalized()
}

// Normalized clamps illegal enum strings. Numeric zeros and false are kept.
func (o Options) Normalized() Options {
	d := Default()
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
	switch o.ObjectWrap {
	case "preserve", "collapse":
	default:
		o.ObjectWrap = d.ObjectWrap
	}
	if o.PrintWidth < 0 {
		o.PrintWidth = d.PrintWidth
	}
	if o.TabWidth < 0 {
		o.TabWidth = d.TabWidth
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

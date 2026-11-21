# fmt

A TypeScript and JavaScript formatter. It parses with [tsgo](https://github.com/startracex/tsgo) and reprints the AST through a width-aware document, in the same spirit as Prettier and oxfmt. It does not call tsgo's language-service formatter (`pkg/format`); that package only adjusts whitespace and will not wrap to a print width.

The output aims at Prettier's defaults (print width 80, indent 2, semicolons, double quotes, trailing commas). It is not a byte-for-byte Prettier or oxfmt match. Differences are listed below.

## Install

Requires Go 1.27 or newer.

```bash
go install github.com/startracex/fmt/cmd/fmt@latest
```

The installed binary is named `fmt`.

## Usage

```bash
fmt .                 # write files in place
fmt src/app.ts        # one file or a directory
fmt --check .         # exit 1 when a file would change
fmt --print-width 100 src
```

With no paths, `fmt` reads stdin and writes stdout. `--stdin-filepath` selects the language and the config lookup directory (default `stdin.ts`).

Formatted extensions: `.ts`, `.tsx`, `.js`, `.jsx`, `.mts`, `.cts`, `.mjs`, `.cjs`.

`node_modules`, `.git`, `dist`, `build`, `coverage`, and `vendor` are skipped. A `.fmtignore` file (gitignore subset: `*`, `?`, `**`, directory rules, `!` negation, `#` comments) is read from the target directory or a parent. `--ignore-path` selects a file explicitly.

## Config

The nearest `.fmtrc.json`, `fmt.config.json`, `.fmtrc`, or `package.json` `"fmt"` object wins. Flags override the file.

```json
{
  "printWidth": 80,
  "tabWidth": 2,
  "useTabs": false,
  "semi": true,
  "singleQuote": false,
  "trailingComma": "all",
  "bracketSpacing": true,
  "arrowParens": "always",
  "endOfLine": "lf",
  "quoteProps": "as-needed"
}
```

`trailingComma` is `all`, `es5`, or `none`. `arrowParens` is `always` or `avoid`. `endOfLine` is `lf`, `crlf`, or `cr`. `quoteProps` is `as-needed`, `consistent`, or `preserve`.

## Library

```go
package main

import (
	"fmt"
	"log"

	"github.com/startracex/fmt/format"
)

func main() {
	out, err := format.Source("app.ts", "const x=1", format.DefaultOptions())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(out)
}
```

## Known differences from Prettier and oxfmt

This is an MVP printer. It will keep growing toward those tools; it does not match them today.

- Wrapping follows print width, but many of Prettier's special cases are missing: hug-style call breaking, member-chain layout, JSX parentheses in `return`, experimental ternaries, and operator-position options.
- Binary and logical chains break before the next line at one indent. A chain that is itself the right-hand side of a broken assignment can pick up an extra indent level.
- Comments are kept and attached from the gaps between tokens. A comment between arguments can move onto the following argument (`f(1 /* c */, 2)` may print as `f(1, /* c */ 2)`). JSDoc is preserved as a comment, not reflowed.
- JSX whitespace is collapsed. Text and elements that fit stay on one line; Prettier's HTML-whitespace and bracket-same-line rules are not implemented.
- Explicit parentheses in the AST are preserved. Prettier drops parentheses that precedence does not require.
- Strings are reprinted with the configured quote and normal escapes. Template text is copied from the source. Numbers, bigints, and regular expressions are copied unchanged.
- `semi: false` omits semicolons except where the next statement starts with `(`, `[`, `` ` ``, `+`, `-`, or `/`. That is a smaller ASI guard than Prettier's.
- Type-literal members use semicolons when `semi` is on, including a semicolon before `}`.
- There is no range formatting, cursor offset, plugin API, or formatting for JSON, CSS, HTML, Markdown, Vue, or Svelte.
- Files with syntax errors are reported and left unchanged.

## License

MIT. See [LICENSE](LICENSE).

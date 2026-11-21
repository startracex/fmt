# fmt

A TypeScript and JavaScript formatter. It parses with [tsgo](https://github.com/startracex/tsgo) and reprints the AST through a width-aware document, in the same spirit as Prettier and oxfmt. The printer does not call tsgo's language-service formatter. That API only tweaks whitespace and will not wrap to a print width.

The library is `pkg/format`. The command is `cmd/fmt`. Defaults match Prettier (print width 80, indent 2, semicolons, double quotes, trailing commas). Remaining differences are listed below.

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

Options are resolved in this order. A later layer replaces a field only when that layer sets it. An explicit `0` or `false` is a value; an omitted field is not.

1. Built-in defaults
2. EditorConfig for that file
3. The nearest fmt config file
4. CLI flags that were actually passed

The fmt config is the nearest `.fmtrc.json`, `fmt.config.json`, `.fmtrc`, or `package.json` `"fmt"` object.

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

EditorConfig (`.editorconfig`, walking parents until `root = true`) fills any fmt option it can speak for and that the fmt config left unset:

| EditorConfig | fmt |
| --- | --- |
| `max_line_length` | `printWidth` (`off` is ignored) |
| `indent_style` | `useTabs` |
| `indent_size` | `tabWidth`, or `useTabs` when the value is `tab` |
| `tab_width` | `tabWidth` (wins over `indent_size`) |
| `end_of_line` | `endOfLine` |
| `quote_type` | `singleQuote` (`single` or `double`; `auto` is ignored) |

## Library

```go
package main

import (
	"fmt"
	"log"

	"github.com/startracex/fmt/pkg/format"
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

Imports, classes, control flow, objects, unions, enums, and JSX that contain elements follow Prettier's usual print-width layout, including parentheses around a broken JSX expression. These still differ:

- A comment between arguments can move onto the following argument (`f(1 /* c */, 2)` may print as `f(1, /* c */ 2)`). JSDoc is kept as a comment and is not reflowed.
- JSX text whitespace is collapsed to single spaces. `htmlWhitespaceSensitivity` and `bracketSameLine` are not implemented.
- Parentheses that are present in the source are kept, other than the ones this printer adds around broken JSX. Prettier also drops parentheses that precedence does not require.
- Strings are reprinted with the configured quotes. Numbers, bigints, regular expressions, and template text are copied from the source.
- `semi: false` inserts a semicolon only when the next statement starts with `(`, `[`, `` ` ``, `+`, `-`, or `/`.
- There is no range formatting, cursor offset, plugin API, or formatting for JSON, CSS, HTML, Markdown, Vue, or Svelte.
- Files with syntax errors are reported and left unchanged.

## License

MIT. See [LICENSE](LICENSE).

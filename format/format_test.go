package format

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCorpus(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"let", "let answer = 42\n"},
		{"const", "const value = 1;\n"},
		{"binary", "const value = a + b + c * d;\n"},
		{"long", "const value = someVeryLongCondition && anotherVeryLongCondition && aThirdVeryLongCondition;\n"},
		{"assign", "const value = someVeryLongCall(veryLongArgument, anotherVeryLongArgument);\n"},
		{"call", "foo(1, 2, 3);\n"},
		{"obj", "const o = { a: 1, b: 2, c: 3 };\n"},
		{"objlong", "const o = { alpha: 1, beta: 2, gamma: 3, delta: 4, epsilon: 5 };\n"},
		{"arr", "const xs = [1, 2, 3];\n"},
		{"fn", "function add(a: number, b: number): number {\n  return a + b;\n}\n"},
		{"arrow", "const add = (a: number, b: number): number => a + b;\n"},
		{"arrowblock", "const add = (a: number, b: number) => {\n  return a + b;\n};\n"},
		{"class", "class Box<T> extends Base implements Foo {\n  value: T;\n  constructor(public value2: T) {\n    super();\n  }\n  get x() {\n    return this.value;\n  }\n  method(a: T) {\n    return a;\n  }\n}\n"},
		{"iface", "interface User {\n  name: string;\n  age?: number;\n  greet(x: string): void;\n}\n"},
		{"type", "type ID = string | number | boolean;\n"},
		{"import", "import { readFile as read, type Stats } from \"node:fs\";\nimport * as path from \"node:path\";\n"},
		{"export", "export function hello(name: string) {\n  return \"hi \" + name;\n}\nexport type { ID } from \"./types\";\nexport * as ns from \"./mod\";\n"},
		{"jsx", "const el = <div className=\"box\" id={id}>hi <b>there</b></div>;\n"},
		{"comment", "// leading\nconst a = 1; // trailing\n\n/* block */\nconst b = 2;\n"},
		{"for", "for (let i = 0; i < 10; i++) {\n  console.log(i);\n}\nfor (const x of xs) {\n  use(x);\n}\n"},
		{"if", "if (cond) {\n  go();\n} else if (other) {\n  stop();\n} else {\n  wait();\n}\n"},
		{"switch", "switch (kind) {\n  case \"a\":\n    run();\n    break;\n  default:\n    halt();\n}\n"},
		{"try", "try {\n  work();\n} catch (e) {\n  fail(e);\n} finally {\n  done();\n}\n"},
		{"template", "const s = `hello ${name}!`;\n"},
		{"async", "async function load(url: string) {\n  const res = await fetch(url);\n  return res.json();\n}\n"},
		{"enum", "enum Color {\n  Red = 1,\n  Blue,\n}\n"},
		{"mapped", "type Rec<T> = { readonly [K in keyof T]?: T[K] };\n"},
		{"generic", "function id<T extends string>(value: T): T {\n  return value;\n}\n"},
		{"chain", "obj.foo().bar().baz();\n"},
		{"new", "const date = new Date();\n"},
		{"destr", "const { a, b: c, ...rest } = obj;\nconst [x, y = 1, ...zs] = arr;\n"},
		{"params", "function f(a: number, b = 1, ...rest: string[]) {\n  return a;\n}\n"},
		{"shebang", "#!/usr/bin/env node\nconsole.log(\"hi\");\n"},
		{"mts", ""},
	}
	show := os.Getenv("FMT_SHOW") != ""
	for _, tc := range cases {
		if tc.src == "" {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			name := "file.ts"
			if tc.name == "jsx" {
				name = "file.tsx"
			}
			out, err := Source(name, tc.src, DefaultOptions())
			if err != nil {
				t.Fatalf("format: %v", err)
			}
			if show {
				t.Logf("\n%s", out)
			}
			if strings.TrimSpace(out) == "" {
				t.Fatalf("empty output")
			}
			again, err := Source(name, out, DefaultOptions())
			if err != nil {
				t.Fatalf("reparse: %v\n%s", err, out)
			}
			if again != out {
				t.Fatalf("not idempotent\nfirst:\n%s\nsecond:\n%s", out, again)
			}
		})
	}
}

func TestFixtures(t *testing.T) {
	root := filepath.Join("..", "testdata", "fixtures")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("no fixtures")
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			dir := filepath.Join(root, e.Name())
			in, err := os.ReadFile(filepath.Join(dir, "input.ts"))
			if err != nil {
				in, err = os.ReadFile(filepath.Join(dir, "input.tsx"))
				if err != nil {
					t.Fatal(err)
				}
			}
			name := "input.ts"
			if _, err := os.Stat(filepath.Join(dir, "input.tsx")); err == nil {
				name = "input.tsx"
			}
			want, err := os.ReadFile(filepath.Join(dir, "output.ts"))
			if err != nil {
				want, err = os.ReadFile(filepath.Join(dir, "output.tsx"))
				if err != nil {
					t.Fatal(err)
				}
			}
			got, err := Source(name, string(in), DefaultOptions())
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				t.Fatalf("fixture mismatch\n got:\n%s\nwant:\n%s", got, want)
			}
			again, err := Source(name, got, DefaultOptions())
			if err != nil {
				t.Fatal(err)
			}
			if again != got {
				t.Fatalf("fixture is not idempotent\n%s", again)
			}
		})
	}
}

func TestSyntaxError(t *testing.T) {
	_, err := Source("bad.ts", "let = ;\n", DefaultOptions())
	if err == nil {
		t.Fatal("expected a syntax error")
	}
}

func TestSingleQuoteAndNoSemi(t *testing.T) {
	opts := DefaultOptions()
	opts.SingleQuote = true
	opts.Semi = false
	got, err := Source("a.ts", "const s = \"hi\";\nconst t = 1\n", opts)
	if err != nil {
		t.Fatal(err)
	}
	want := "const s = 'hi'\nconst t = 1\n"
	if got != want {
		t.Fatalf("got %q", got)
	}
}

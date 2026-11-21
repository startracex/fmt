package format

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func userOptions() Options {
	o := DefaultOptions()
	o.PrintWidth = 100
	o.TabWidth = 2
	o.UseTabs = false
	o.Semi = true
	o.SingleQuote = false
	o.QuoteProps = "as-needed"
	o.ArrowParens = "always"
	o.TrailingComma = "all"
	o.BracketSpacing = true
	o.ObjectWrap = "preserve"
	return o
}

func mustFormat(t *testing.T, name, src string, opts Options) string {
	t.Helper()
	out, err := Source(name, src, opts)
	if err != nil {
		t.Fatalf("format: %v\n%s", err, src)
	}
	again, err := Source(name, out, opts)
	if err != nil {
		t.Fatalf("reformat: %v\n%s", err, out)
	}
	if again != out {
		t.Fatalf("not idempotent\nfirst:\n%s\nsecond:\n%s", out, again)
	}
	return out
}

func TestBraceStaysWithEquals(t *testing.T) {
	src := "const baseDefaults: Omit<SWRConfig, \"requestUpdate\"> = {\n" +
		"  refreshInterval: 0,\n" +
		"  maxAge: 5000,\n" +
		"  dedupingInterval: 2000,\n" +
		"  revalidateOnFocus: true,\n" +
		"  isPaused: () => false,\n" +
		"};\n"
	out := mustFormat(t, "config.ts", src, userOptions())
	if !strings.Contains(out, "const baseDefaults: Omit<SWRConfig, \"requestUpdate\"> = {") {
		t.Fatalf("opening brace left the equals line:\n%s", out)
	}
	if strings.Contains(out, "=\n") {
		t.Fatalf("assignment broke before the brace:\n%s", out)
	}

	flat := "const baseDefaults: Omit<SWRConfig, \"requestUpdate\"> = { refreshInterval: 0, maxAge: 5000, dedupingInterval: 2000, revalidateOnFocus: true, revalidateOnReconnect: true };\n"
	out = mustFormat(t, "config.ts", flat, userOptions())
	if !strings.Contains(out, "const baseDefaults: Omit<SWRConfig, \"requestUpdate\"> = {") {
		t.Fatalf("flat object dropped the brace:\n%s", out)
	}
}

func TestObjectWrapPreserveOverrides(t *testing.T) {
	src := "import config from \"@startracex/dev-config/oxfmt\";\n" +
		"import { defineConfig } from \"oxfmt\";\n" +
		"\n" +
		"export default defineConfig({\n" +
		"  ...config,\n" +
		"  overrides: [\n" +
		"    {\n" +
		"      files: [\"**/*.json\"],\n" +
		"      options: {\n" +
		"        printWidth: 1,\n" +
		"      },\n" +
		"    },\n" +
		"  ],\n" +
		"});\n"
	out := mustFormat(t, "oxfmt.config.ts", src, userOptions())
	if out != src {
		t.Fatalf("expanded overrides were rewritten:\n%s", out)
	}

	flat := "export default defineConfig({ ...config, overrides: [{ files: [\"**/*.json\"], options: { printWidth: 1 } }] });\n"
	out = mustFormat(t, "oxfmt.config.ts", flat, userOptions())
	want := "export default defineConfig({\n" +
		"  ...config,\n" +
		"  overrides: [{ files: [\"**/*.json\"], options: { printWidth: 1 } }],\n" +
		"});\n"
	if out != want {
		t.Fatalf("flat overrides:\n%s", out)
	}

	collapsed := userOptions()
	collapsed.ObjectWrap = "collapse"
	short := "const point = {\n  x: 1,\n  y: 2,\n};\n"
	out = mustFormat(t, "point.ts", short, collapsed)
	if out != "const point = { x: 1, y: 2 };\n" {
		t.Fatalf("collapse should pack a short object:\n%s", out)
	}
	out = mustFormat(t, "point.ts", short, userOptions())
	if out != short {
		t.Fatalf("preserve should keep the short object expanded:\n%s", out)
	}

	array := "const xs = [\n  1,\n  2,\n];\n"
	out = mustFormat(t, "xs.ts", array, userOptions())
	if out != array {
		t.Fatalf("preserve should keep the array expanded:\n%s", out)
	}
	out = mustFormat(t, "xs.ts", array, collapsed)
	if out != "const xs = [1, 2];\n" {
		t.Fatalf("collapse should pack the array:\n%s", out)
	}
}

func TestTimersSetStaysBroken(t *testing.T) {
	// The flat form is under print width 100. Prettier still breaks it because
	// one argument is a call that takes a function.
	src := "class RefreshScheduler {\n" +
		"  private scheduleRefresh(host: ReactiveControllerHost) {\n" +
		"    this.timers.set(host, setInterval(() => this.refreshHost(host), refreshInterval));\n" +
		"  }\n" +
		"}\n"
	want := "class RefreshScheduler {\n" +
		"  private scheduleRefresh(host: ReactiveControllerHost) {\n" +
		"    this.timers.set(\n" +
		"      host,\n" +
		"      setInterval(() => this.refreshHost(host), refreshInterval),\n" +
		"    );\n" +
		"  }\n" +
		"}\n"
	out := mustFormat(t, "refresh-scheduler.ts", src, userOptions())
	if out != want {
		t.Fatalf("timers.set:\n%s", out)
	}
}

func TestDescribeDoesNotOverwrap(t *testing.T) {
	src := "describe(\"abort / signal\", () => {\n" +
		"  beforeEach(() => {\n" +
		"    resetLibraryState();\n" +
		"  });\n" +
		"\n" +
		"  afterEach(() => {\n" +
		"    resetLibraryState();\n" +
		"  });\n" +
		"\n" +
		"  it(\"passes AbortSignal to the fetcher\", async () => {\n" +
		"    const key = uniqueKey(\"signal\");\n" +
		"  });\n" +
		"});\n"
	out := mustFormat(t, "abort.test.ts", src, userOptions())
	for _, header := range []string{
		"describe(\"abort / signal\", () => {",
		"  beforeEach(() => {",
		"  afterEach(() => {",
		"  it(\"passes AbortSignal to the fetcher\", async () => {",
	} {
		if !strings.Contains(out, header) {
			t.Fatalf("missing %s\n%s", header, out)
		}
	}
	if strings.Contains(out, "describe(\n") || strings.Contains(out, "beforeEach(\n") || strings.Contains(out, "it(\n") {
		t.Fatalf("call header was wrapped:\n%s", out)
	}
}

func TestCallbackInTheMiddleBreaksTheCall(t *testing.T) {
	src := "const controller = new SWRController(\n" +
		"  createHost(),\n" +
		"  key,\n" +
		"  (_k, init) => {\n" +
		"    return \"ok\";\n" +
		"  },\n" +
		"  quietConfig,\n" +
		");\n"
	out := mustFormat(t, "abort.test.ts", src, userOptions())
	if out != src {
		t.Fatalf("call with a middle callback:\n%s", out)
	}
	packed := "const controller = new SWRController(createHost(), key, (_k, init) => {\n  return \"ok\";\n}, quietConfig);\n"
	out = mustFormat(t, "abort.test.ts", packed, userOptions())
	if out != src {
		t.Fatalf("packed call should expand:\n%s", out)
	}
}

func TestDecoratorBeforeExport(t *testing.T) {
	cases := []struct{ src, want string }{
		{
			"@dec()\nexport class Foo {}\n",
			"@dec()\nexport class Foo {}\n",
		},
		{
			"@dec()\nexport default class Foo {}\n",
			"@dec()\nexport default class Foo {}\n",
		},
		{
			"@dec()\nexport default abstract class Foo {}\n",
			"@dec()\nexport default abstract class Foo {}\n",
		},
		{
			"@dec()\nabstract class Foo {}\n",
			"@dec()\nabstract class Foo {}\n",
		},
		{
			"@dec()\nclass Foo {}\n",
			"@dec()\nclass Foo {}\n",
		},
		{
			"@dec()\nexport function f() {}\n",
			"@dec()\nexport function f() {}\n",
		},
		{
			"@dec()\nexport abstract class Foo {}\n",
			"@dec()\nexport abstract class Foo {}\n",
		},
		{
			"class A {\n  @dec()\n  method() {}\n  @dec()\n  field = 1;\n}\n",
			"class A {\n  @dec()\n  method() {}\n  @dec()\n  field = 1;\n}\n",
		},
	}
	for _, tc := range cases {
		out := mustFormat(t, "dec.ts", tc.src, DefaultOptions())
		if out != tc.want {
			t.Fatalf("decorator spacing\nsrc:\n%sgot:\n%swant:\n%s", tc.src, out, tc.want)
		}
		if strings.Contains(out, "\n export") || strings.Contains(out, "\n class") || strings.Contains(out, "\n abstract") || strings.Contains(out, "\n function") {
			t.Fatalf("leading space after decorator:\n%s", out)
		}
	}
}

func TestArrowBreaksAfterFatArrow(t *testing.T) {
	src := "export const isDisabledKey = (key: unknown): boolean => key === null || key === undefined || key === false;\n"
	want := "export const isDisabledKey = (key: unknown): boolean =>\n" +
		"  key === null || key === undefined || key === false;\n"
	out := mustFormat(t, "config.ts", src, userOptions())
	if out != want {
		t.Fatalf("arrow assignment:\n%s", out)
	}
}

func TestBinaryParensAndComments(t *testing.T) {
	opts := DefaultOptions()
	opts.PrintWidth = 80
	src := "!(x || y);\n(a && b) || c;\na && b || c;\na + (b + c);\n(a + b) % c;\na & (b + c);\na == 3 ? (a = 1) : (a = 2);\n"
	out := mustFormat(t, "bit.js", src, opts)
	want := "!(x || y);\n(a && b) || c;\n(a && b) || c;\na + b + c;\n(a + b) % c;\na & (b + c);\na == 3 ? (a = 1) : (a = 2);\n"
	if out != want {
		t.Fatalf("parens:\n%s", out)
	}

	src = "!(x || // foo\n  y || // bar\n  z);\n"
	out = mustFormat(t, "c.js", src, opts)
	if !strings.Contains(out, "x || // foo") || !strings.Contains(out, "y || // bar") {
		t.Fatalf("comment drifted off the operator:\n%s", out)
	}

	src = "const numbers = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25];\n"
	out = mustFormat(t, "n.js", src, opts)
	if !strings.Contains(out, "1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22,") {
		t.Fatalf("numbers were not packed:\n%s", out)
	}
}

func TestTernaryChainLayout(t *testing.T) {
	src := "const value = condition1\n? value1\n: condition2\n    ? value2\n    : condition3\n        ? value3\n        : value4;\n\nconst y = A ? B ? C : D : E;\n\nconst z = verylongtestexpressionthatdoesnotfitonthelineatallreallyreallylong\n  ? alsoaverylongconsequentthatisitselfaternary ? yes : no\n  : alternate;\n"
	opts := DefaultOptions()
	opts.PrintWidth = 80
	opts.TabWidth = 2
	out := mustFormat(t, "tern.js", src, opts)
	want := "const value = condition1\n" +
		"  ? value1\n" +
		"  : condition2\n" +
		"    ? value2\n" +
		"    : condition3\n" +
		"      ? value3\n" +
		"      : value4;\n" +
		"\n" +
		"const y = A ? (B ? C : D) : E;\n" +
		"\n" +
		"const z = verylongtestexpressionthatdoesnotfitonthelineatallreallyreallylong\n" +
		"  ? alsoaverylongconsequentthatisitselfaternary\n" +
		"    ? yes\n" +
		"    : no\n" +
		"  : alternate;\n"
	if out != want {
		t.Fatalf("tabWidth 2:\n%s", out)
	}

	opts.TabWidth = 4
	out = mustFormat(t, "tern.js", src, opts)
	want = "const value = condition1\n" +
		"    ? value1\n" +
		"    : condition2\n" +
		"      ? value2\n" +
		"      : condition3\n" +
		"        ? value3\n" +
		"        : value4;\n" +
		"\n" +
		"const y = A ? (B ? C : D) : E;\n" +
		"\n" +
		"const z = verylongtestexpressionthatdoesnotfitonthelineatallreallyreallylong\n" +
		"    ? alsoaverylongconsequentthatisitselfaternary\n" +
		"        ? yes\n" +
		"        : no\n" +
		"    : alternate;\n"
	if out != want {
		t.Fatalf("tabWidth 4:\n%s", out)
	}
}

func TestPackagesLiveUnderPkg(t *testing.T) {
	root := filepath.Join("..", "..")
	internal := filepath.Join(root, "internal")
	err := filepath.WalkDir(internal, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".go") {
			t.Errorf("library file still under internal/: %s", path)
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	for _, pkg := range []string{"format", "printer", "cli", "config", "editorconfig", "ignore", "opt"} {
		dir := filepath.Join(root, "pkg", pkg)
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Errorf("pkg/%s: %v", pkg, err)
			continue
		}
		var goFiles int
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), "_test.go") {
				goFiles++
			}
		}
		if goFiles == 0 {
			t.Errorf("pkg/%s has no implementation files", pkg)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "cmd", "fmt", "main.go")); err != nil {
		t.Fatal(err)
	}
}

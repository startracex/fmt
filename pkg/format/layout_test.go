package format

import "testing"

func TestIntersectionAlias(t *testing.T) {
	src := "export type Foo = Bar & Baz & Qux;\ntype Pair = { a: number } & { b: string };\ntype U = A | B;\n"
	out, err := Source("t.ts", src, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	want := "export type Foo = Bar & Baz & Qux;\ntype Pair = { a: number } & { b: string };\ntype U = A | B;\n"
	if out != want {
		t.Fatalf("got:\n%s", out)
	}
}

func TestObjectArrayPreserveAndHug(t *testing.T) {
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
	out, err := Source("oxfmt.config.ts", src, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if out != src {
		t.Fatalf("expanded config was rewritten:\n%s", out)
	}

	flat := "export default defineConfig({ ...config, overrides: [{ files: [\"**/*.json\"], options: { printWidth: 1 } }] });\n"
	out, err = Source("oxfmt.config.ts", flat, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	want := "export default defineConfig({\n" +
		"  ...config,\n" +
		"  overrides: [{ files: [\"**/*.json\"], options: { printWidth: 1 } }],\n" +
		"});\n"
	if out != want {
		t.Fatalf("flat config:\n%s", out)
	}
}

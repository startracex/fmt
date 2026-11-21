package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteAndCheck(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "main.ts")
	if err := os.WriteFile(src, []byte("const x=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ignored := filepath.Join(dir, "node_modules", "pkg.ts")
	if err := os.MkdirAll(filepath.Dir(ignored), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ignored, []byte("const y=2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	skip := filepath.Join(dir, "skip.ts")
	if err := os.WriteFile(skip, []byte("const z=3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".fmtignore"), []byte("skip.ts\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := Main([]string{dir}, &stdout, &stderr); code != 0 {
		t.Fatalf("write code %d\n%s\n%s", code, stdout.String(), stderr.String())
	}
	got, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "const x = 1;\n" {
		t.Fatalf("formatted: %q", got)
	}
	raw, _ := os.ReadFile(ignored)
	if string(raw) != "const y=2\n" {
		t.Fatalf("node_modules was formatted: %q", raw)
	}
	raw, _ = os.ReadFile(skip)
	if string(raw) != "const z=3\n" {
		t.Fatalf("ignored file was formatted: %q", raw)
	}

	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"--check", dir}, &stdout, &stderr); code != 0 {
		t.Fatalf("clean check code %d\n%s", code, stderr.String())
	}

	if err := os.WriteFile(src, []byte("const x=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if code := Main([]string{"--check", src}, &stdout, &stderr); code != 1 {
		t.Fatalf("dirty check code %d stdout %q", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "main.ts") {
		t.Fatalf("check output %q", stdout.String())
	}
}

func TestFlagOverridesConfigAndEditorConfig(t *testing.T) {
	dir := t.TempDir()
	src := "function f() {\nreturn 1\n}\n"
	path := filepath.Join(dir, "a.ts")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	ec := "root = true\n\n[*]\nindent_style = space\nindent_size = 4\n"
	if err := os.WriteFile(filepath.Join(dir, ".editorconfig"), []byte(ec), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Main([]string{"--write", path}, &stdout, &stderr); code != 0 {
		t.Fatalf("editorconfig code %d\n%s", code, stderr.String())
	}
	got, _ := os.ReadFile(path)
	if string(got) != "function f() {\n    return 1;\n}\n" {
		t.Fatalf("editorconfig indent:\n%s", got)
	}

	if err := os.WriteFile(filepath.Join(dir, ".fmtrc.json"), []byte(`{"tabWidth": 0}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"--write", path}, &stdout, &stderr); code != 0 {
		t.Fatalf("config code %d\n%s", code, stderr.String())
	}
	got, _ = os.ReadFile(path)
	if string(got) != "function f() {\nreturn 1;\n}\n" {
		t.Fatalf("explicit tabWidth 0:\n%s", got)
	}

	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"--write", "--tab-width", "8", path}, &stdout, &stderr); code != 0 {
		t.Fatalf("flag code %d\n%s", code, stderr.String())
	}
	got, _ = os.ReadFile(path)
	if string(got) != "function f() {\n        return 1;\n}\n" {
		t.Fatalf("flag indent:\n%s", got)
	}
}

func TestSyntaxErrorStatus(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.ts")
	if err := os.WriteFile(path, []byte("let =\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Main([]string{path}, &stdout, &stderr); code != 2 {
		t.Fatalf("code %d stderr %s", code, stderr.String())
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "let =\n" {
		t.Fatalf("failed format changed the file: %q", got)
	}
}

func TestIntersectionDoesNotPanic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "types.ts")
	src := "export type Foo = Bar & Baz & Qux;\ntype T = A | B;\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Main([]string{path}, &stdout, &stderr); code != 0 {
		t.Fatalf("code %d stderr %s", code, stderr.String())
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "panic") || !strings.Contains(string(got), "Bar &") {
		t.Fatalf("intersection output:\n%s", got)
	}
}

func TestWriteAtomicReplacesWholeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.ts")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeAtomic(path, []byte("const value = 1;\n")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "const value = 1;\n" {
		t.Fatalf("got %q", got)
	}
	matches, _ := filepath.Glob(filepath.Join(dir, ".fmt-*"))
	if len(matches) != 0 {
		t.Fatalf("temp files left behind: %v", matches)
	}
}

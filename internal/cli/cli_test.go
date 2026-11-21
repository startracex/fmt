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
}

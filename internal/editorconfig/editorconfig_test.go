package editorconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/startracex/fmt/internal/config"
	"github.com/startracex/fmt/internal/opt"
)

func TestAppliesWhenUnset(t *testing.T) {
	dir := t.TempDir()
	ec := "" +
		"root = true\n" +
		"\n" +
		"[*]\n" +
		"indent_style = tab\n" +
		"indent_size = 8\n" +
		"tab_width = 4\n" +
		"end_of_line = crlf\n" +
		"max_line_length = 120\n" +
		"quote_type = single\n" +
		"\n" +
		"[*.ts]\n" +
		"indent_style = space\n" +
		"indent_size = 2\n" +
		"max_line_length = off\n"
	if err := os.WriteFile(filepath.Join(dir, ".editorconfig"), []byte(ec), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(filepath.Join(dir, "app.ts"))
	if err != nil {
		t.Fatal(err)
	}
	resolved := got.Resolve()
	if resolved.UseTabs {
		t.Fatal("indent_style=space should clear tabs")
	}
	if resolved.TabWidth != 4 {
		t.Fatalf("tab_width should win over indent_size, got %d", resolved.TabWidth)
	}
	if resolved.EndOfLine != "crlf" || resolved.PrintWidth != 80 || !resolved.SingleQuote {
		t.Fatalf("%+v", resolved)
	}
	js, err := Load(filepath.Join(dir, "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	jsResolved := js.Resolve()
	if !jsResolved.UseTabs || jsResolved.PrintWidth != 120 || jsResolved.TabWidth != 4 {
		t.Fatalf("js: %+v", jsResolved)
	}
}

func TestConfigOverridesEditorConfig(t *testing.T) {
	dir := t.TempDir()
	ec := "root = true\n\n[*]\nindent_style = tab\nindent_size = 8\nmax_line_length = 100\nend_of_line = crlf\n"
	if err := os.WriteFile(filepath.Join(dir, ".editorconfig"), []byte(ec), 0o644); err != nil {
		t.Fatal(err)
	}
	fmtrc := `{"printWidth": 0, "useTabs": false, "tabWidth": 0}`
	if err := os.WriteFile(filepath.Join(dir, ".fmtrc.json"), []byte(fmtrc), 0o644); err != nil {
		t.Fatal(err)
	}
	editor, err := Load(filepath.Join(dir, "a.ts"))
	if err != nil {
		t.Fatal(err)
	}
	file, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	resolved := editor.Merge(file).Resolve()
	if resolved.PrintWidth != 0 || resolved.UseTabs || resolved.TabWidth != 0 {
		t.Fatalf("explicit config zeros lost: %+v", resolved)
	}
	if resolved.EndOfLine != "crlf" {
		t.Fatalf("unset endOfLine should keep editorconfig, got %q", resolved.EndOfLine)
	}
}

func TestRootStopsParent(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "pkg")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, ".editorconfig"), []byte("[*]\nindent_size = 8\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(child, ".editorconfig"), []byte("root = true\n\n[*]\nindent_size = 3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(filepath.Join(child, "a.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if got.TabWidth == nil || *got.TabWidth != 3 {
		t.Fatalf("%+v", got)
	}
}

func TestNoEditorConfig(t *testing.T) {
	dir := t.TempDir()
	got, err := Load(filepath.Join(dir, "a.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if got != (opt.Partial{}) {
		t.Fatalf("%+v", got)
	}
}

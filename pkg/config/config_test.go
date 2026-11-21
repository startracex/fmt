package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/startracex/fmt/pkg/opt"
)

func TestExplicitZeroIsSet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".fmtrc.json")
	body := `{"printWidth": 0, "tabWidth": 0, "useTabs": false, "semi": false}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.PrintWidth == nil || *got.PrintWidth != 0 || got.TabWidth == nil || *got.TabWidth != 0 {
		t.Fatalf("zeros were dropped: %+v", got)
	}
	if got.UseTabs == nil || *got.UseTabs || got.Semi == nil || *got.Semi {
		t.Fatalf("false was dropped: %+v", got)
	}
	resolved := got.Resolve()
	if resolved.PrintWidth != 0 || resolved.TabWidth != 0 || resolved.UseTabs || resolved.Semi {
		t.Fatalf("resolve rewrote explicit zeros: %+v", resolved)
	}
}

func TestUnsetKeepsLowerLayer(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".fmtrc.json"), []byte(`{"singleQuote": true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	file, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if file.PrintWidth != nil || file.Semi != nil || file.TabWidth != nil {
		t.Fatalf("unset fields should stay nil: %+v", file)
	}
	lower := opt.Partial{PrintWidth: intPtr(100), Semi: boolPtr(false), TabWidth: intPtr(8)}
	resolved := lower.Merge(file).Resolve()
	if resolved.PrintWidth != 100 || resolved.Semi || resolved.TabWidth != 8 || !resolved.SingleQuote {
		t.Fatalf("%+v", resolved)
	}
}

func TestEmptyObjectUsesDefaults(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "fmt.config.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	resolved := got.Resolve()
	def := opt.Default()
	if resolved != def {
		t.Fatalf("got %+v want %+v", resolved, def)
	}
}

func intPtr(n int) *int    { return &n }
func boolPtr(v bool) *bool { return &v }

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/startracex/fmt/internal/opt"
)

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".fmtrc.json")
	if err := os.WriteFile(path, []byte(`{"printWidth": 100, "singleQuote": true, "semi": false}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir, opt.Default())
	if err != nil {
		t.Fatal(err)
	}
	if got.PrintWidth != 100 || !got.SingleQuote || got.Semi {
		t.Fatalf("%+v", got)
	}
}

package ignore

import "testing"

func TestParse(t *testing.T) {
	set, err := Parse("# comment\nskip.ts\n!keep.ts\ndist/\n")
	if err != nil {
		t.Fatal(err)
	}
	if !set.Match("skip.ts", false) {
		t.Fatal("skip.ts should match")
	}
	if set.Match("keep.ts", false) {
		t.Fatal("negation should un-ignore keep.ts")
	}
	if !set.Match("pkg/skip.ts", false) {
		t.Fatal("unanchored pattern should match in a subdirectory")
	}
	if !set.Match("dist", true) || set.Match("dist", false) {
		t.Fatal("directory-only pattern")
	}
	if !Always("src/node_modules/pkg/index.js") {
		t.Fatal("node_modules should always be skipped")
	}
}

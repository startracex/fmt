package doc

import "testing"

func TestFlatAndBreak(t *testing.T) {
	d := Group(Concat(T("hello"), Line(), T("world")))
	if got := Print(d, PrintOptions{Width: 80, IndentUnit: "  "}); got != "hello world" {
		t.Fatalf("flat: %q", got)
	}
	if got := Print(d, PrintOptions{Width: 5, IndentUnit: "  "}); got != "hello\nworld" {
		t.Fatalf("break: %q", got)
	}
}

func TestIndentWhenBroken(t *testing.T) {
	d := Group(Concat(T("{"), Indent(Concat(HardLine(), T("a"))), HardLine(), T("}")))
	got := Print(d, PrintOptions{Width: 80, IndentUnit: "  "})
	want := "{\n  a\n}"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestFlatParentDoesNotDoubleIndent(t *testing.T) {
	// A flat assignment should not add an indent level around a broken block.
	block := Group(Concat(T("{"), Indent(Concat(HardLine(), T("return"))), HardLine(), T("}")))
	d := Group(Concat(T("const f = () =>"), Indent(Concat(Line(), block))))
	got := Print(d, PrintOptions{Width: 80, IndentUnit: "  "})
	want := "const f = () => {\n  return\n}"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

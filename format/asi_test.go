package format

import "testing"

func TestASI(t *testing.T) {
	opts := DefaultOptions()
	opts.Semi = false
	got, err := Source("a.ts", "a = b\n;(function () {})()\n", opts)
	if err != nil {
		t.Fatal(err)
	}
	want := "a = b;\n(function () {})()\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

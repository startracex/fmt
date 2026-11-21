package format

import "testing"

func TestRootedFileName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{`C:\source\lit-swr\oxfmt.config.ts`, `C:/source/lit-swr/oxfmt.config.ts`},
		{`C:/source/lit-swr/oxfmt.config.ts`, `C:/source/lit-swr/oxfmt.config.ts`},
		{`c:\source\a.ts`, `c:/source/a.ts`},
		{`C:\foo\..\bar.ts`, `C:/bar.ts`},
		{`C:\source/lit-swr\file.ts`, `C:/source/lit-swr/file.ts`},
		{`\\server\share\a.ts`, `//server/share/a.ts`},
		{`//server/share/a.ts`, `//server/share/a.ts`},
		{`file:///C:/source/a.ts`, `file:///C:/source/a.ts`},
		{`/home/a.ts`, `/home/a.ts`},
		{`src\app.ts`, `/src/app.ts`},
		{`src/app.ts`, `/src/app.ts`},
		{`foo\bar/baz.ts`, `/foo/bar/baz.ts`},
		{`.\app.ts`, `/app.ts`},
		{`./app.ts`, `/app.ts`},
		{`..\app.ts`, `/app.ts`},
		{`../app.ts`, `/app.ts`},
		{`file.ts`, `/file.ts`},
		{``, `/file.ts`},
		// Drive-relative: no per-drive working directory is available.
		{`C:foo.ts`, `/C:foo.ts`},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := rootedFileName(tc.in).AsString()
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestSourceWindowsPath(t *testing.T) {
	// The reported panic: formatting C:\source\lit-swr\oxfmt.config.ts.
	out, err := Source(`C:\source\lit-swr\oxfmt.config.ts`, "const x=1\n", DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if out != "const x = 1;\n" {
		t.Fatalf("got %q", out)
	}
	tsx, err := Source(`C:\source\app.tsx`, "const el = <div />;\n", DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if tsx != "const el = <div />;\n" {
		t.Fatalf("tsx got %q", tsx)
	}
}

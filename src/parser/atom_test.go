package parser_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"Atom3/src/parser"
)

// normalize makes comparisons independent of line endings, trailing blank
// lines, and which path separator the test ran on.
func normalize(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, `\`, "/")
	return strings.TrimRight(s, "\n")
}

// TestPrograms runs every program in testdata and compares its output with the
// matching .expected file.
func TestPrograms(t *testing.T) {
	programs, err := filepath.Glob(filepath.Join("testdata", "*.atom"))
	if err != nil {
		t.Fatal(err)
	}
	if len(programs) == 0 {
		t.Fatal("no programs found in testdata")
	}

	for _, path := range programs {
		t.Run(filepath.Base(path), func(t *testing.T) {
			want, err := os.ReadFile(strings.TrimSuffix(path, ".atom") + ".expected")
			if err != nil {
				t.Fatalf("no expected output: %v", err)
			}

			var out bytes.Buffer
			parser.Output = &out
			t.Cleanup(func() { parser.Output = os.Stdout })

			parser.ResetAll()
			parser.RunFile(path)

			got, expected := normalize(out.String()), normalize(string(want))
			if got != expected {
				t.Errorf("output mismatch\n--- got ---\n%s\n--- want ---\n%s", got, expected)
			}
		})
	}
}

func run(t *testing.T, src string) string {
	t.Helper()

	var out bytes.Buffer
	parser.Output = &out
	t.Cleanup(func() { parser.Output = os.Stdout })

	parser.ResetAll()
	parser.RunSource(src)
	return normalize(out.String())
}

// Nesting was broken three separate ways before blocks were compiled into a
// tree, so each shape is pinned here.
func TestNesting(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "if inside if resumes the outer block",
			src:  "if [true] {\nif [false] {\nprint(\"no\")\n}\nprint(\"yes\")\n}",
			want: "yes",
		},
		{
			name: "skipped block does not run its inner block",
			src:  "if [false] {\nif [true] {\nprint(\"no\")\n}\n}\nprint(\"after\")",
			want: "after",
		},
		{
			name: "while inside while",
			src:  "var a = 0\nwhile [a < 2] {\nvar b = 0\nwhile [b < 2] {\nprint(\"x\")\nvar b = [b + 1]\n}\nvar a = [a + 1]\n}",
			want: "x\nx\nx\nx",
		},
		{
			name: "while inside a false if never runs",
			src:  "if [false] {\nwhile [true] {\nprint(\"no\")\n}\n}\nprint(\"after\")",
			want: "after",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := run(t, c.src); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

// Parameters used to be globals, which made recursion silently wrong.
func TestCallFramesAreIsolated(t *testing.T) {
	src := "var n = \"global\"\nfunc f(n) {\nreturn n\n}\nprint([f(\"local\")])\nprint(n)"

	if got := run(t, src); got != "local\nglobal" {
		t.Errorf("got %q, want %q", got, "local\nglobal")
	}
}

// Nested brackets truncated at the first ] while the grammar was regex based.
func TestNestedBracketsSurviveParsing(t *testing.T) {
	src := "var cfg = [host: \"h\", ports: [80, 443]]\nprint([(cfg @ \"ports\") @ 1])"

	if got := run(t, src); got != "443" {
		t.Errorf("got %q, want %q", got, "443")
	}
}

func TestParseErrorsReportLineNumbers(t *testing.T) {
	got := run(t, "print(\"a\")\nvar broken\nprint(\"b\")")

	if !strings.Contains(got, "line 2") {
		t.Errorf("error should name line 2, got %q", got)
	}
	if !strings.Contains(got, "a") || !strings.Contains(got, "b") {
		t.Errorf("a bad line should not stop the rest of the program, got %q", got)
	}
}

package expr

import (
	"testing"
	"unsafe"
)

// Value must stay four words wide. At five, Go stops returning it in registers
// and the interpreter measured roughly three times slower, so this guards the
// size rather than trusting anyone to remember why.
func TestValueStaysFourWords(t *testing.T) {
	const maxWords = 4

	if size := unsafe.Sizeof(Value{}); size > maxWords*unsafe.Sizeof(uintptr(0)) {
		t.Errorf("Value grew to %d bytes; keep it at %d or the interpreter slows down sharply",
			size, maxWords*unsafe.Sizeof(uintptr(0)))
	}
}

func TestUnescape(t *testing.T) {
	cases := map[string]string{
		"plain":        "plain",
		`a\nb`:         "a\nb",
		`a\tb`:         "a\tb",
		`a\rb`:         "a\rb",
		`say \"hi\"`:   `say "hi"`,
		`back\\slash`:  `back\slash`,
		`C:\Users\Fry`: `C:\Users\Fry`,
	}

	for in, want := range cases {
		if got := Unescape(in); got != want {
			t.Errorf("Unescape(%q) = %q, want %q", in, got, want)
		}
	}
}

func evalString(t *testing.T, src string) string {
	t.Helper()

	scope := NewScope()

	e, err := Compile(src, scope)
	if err != nil {
		t.Fatalf("Compile(%q): %v", src, err)
	}
	return e(scope).Display()
}

func TestCompileAndEval(t *testing.T) {
	cases := map[string]string{
		"1 + 2 * 3":         "7",
		"(1 + 2) * 3":       "9",
		"10 / 4":            "2.5",
		"7 % 4":             "3",
		"-3 + 5":            "2",
		"2 < 3":             "true",
		"3 <= 3":            "true",
		"2 == 2":            "true",
		"2 ?= 3":            "true",
		"true && false":     "false",
		"false || true":     "true",
		"!false":            "true",
		`"a" + "b"`:         "ab",
		"1, 2, 3":           "[1, 2, 3]",
		"[1, 2] + [3]":      "[1, 2, 3]",
		"[10, 20] @ 1":      "20",
		"k: 1, j: 2":        "[j: 2, k: 1]",
		`["a": 1] @ "a"`:    "1",
		"[1, 2] == [1, 2]":  "true",
		"[1, 2] == [1, 3]":  "false",
		"len: 0":            "[len: 0]",
		"1 + 2 == 3":        "true",
		"[[1, 2], [3]] @ 0": "[1, 2]",
	}

	for src, want := range cases {
		if got := evalString(t, src); got != want {
			t.Errorf("%q evaluated to %q, want %q", src, got, want)
		}
	}
}

func TestCompileRejectsBadInput(t *testing.T) {
	bad := []string{
		"1 +",
		"(1 + 2",
		`"unterminated`,
		"1 2",
		"@ 3",
		"[1, 2",
	}

	for _, src := range bad {
		if _, err := Compile(src, NewScope()); err == nil {
			t.Errorf("Compile(%q) should have failed", src)
		}
	}
}

// Indexing past the end yields an empty value rather than panicking, because a
// script should not be able to crash the interpreter.
func TestOutOfRangeIndexIsSafe(t *testing.T) {
	for _, src := range []string{"[1, 2] @ 9", "[1, 2] @ -1", `[a: 1] @ "missing"`} {
		if got := evalString(t, src); got != "0" && got != "" {
			t.Errorf("%q returned %q, want an empty value", src, got)
		}
	}
}

package parser_test

import (
	"io"
	"os"
	"strings"
	"testing"

	"Atom3/src/parser"
)

// These exist to be profiled, not to produce a headline number. Compilation
// happens once outside the timer so what is measured is execution.
func benchProgram(b *testing.B, src string) {
	b.Helper()

	parser.Output = io.Discard
	b.Cleanup(func() { parser.Output = os.Stdout })

	parser.ResetAll()
	program := parser.Compile(strings.Split(src, "\n"))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		parser.Run(program)
	}
}

func BenchmarkLoop(b *testing.B) {
	benchProgram(b, "var i = 0\nwhile [i < 10000] {\nvar i = [i + 1]\n}")
}

func BenchmarkArith(b *testing.B) {
	benchProgram(b, "var i = 0\nvar t = 0\nwhile [i < 10000] {\nvar t = [t + i * 2]\nvar i = [i + 1]\n}")
}

func BenchmarkBranch(b *testing.B) {
	benchProgram(b, "var i = 0\nwhile [i < 10000] {\nif [i > 5] {\nvar i = [i + 1]\n}\nelse {\nvar i = [i + 1]\n}\n}")
}

func BenchmarkCall(b *testing.B) {
	benchProgram(b, "func add(a, b) {\nreturn [a + b]\n}\nvar i = 0\nvar t = 0\nwhile [i < 10000] {\nvar t = [add(t, i)]\nvar i = [i + 1]\n}")
}

func BenchmarkListIndex(b *testing.B) {
	benchProgram(b, "var xs = [1, 2, 3, 4, 5, 6, 7, 8]\nvar i = 0\nvar t = 0\nwhile [i < 10000] {\nvar t = [t + xs @ 3]\nvar i = [i + 1]\n}")
}

func BenchmarkStringConcat(b *testing.B) {
	benchProgram(b, "var i = 0\nvar s = \"\"\nwhile [i < 2000] {\nvar s = [\"x\" + \"y\"]\nvar i = [i + 1]\n}")
}

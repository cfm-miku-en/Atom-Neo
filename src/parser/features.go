package parser

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// Output is where the interpreter writes. Tests point it at a buffer; the CLI
// leaves it on stdout.
var Output io.Writer = os.Stdout

func errorf(format string, args ...any) {
	fmt.Fprintf(Output, format, args...)
}

func ResolveValue(arg string) string {
	if v, ok := scope.Lookup(arg); ok {
		return v.Display()
	}
	return strings.Trim(arg, `"`)
}

func Print(text string) {
	fmt.Fprintln(Output, text)
}

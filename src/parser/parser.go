package parser

import (
	"bufio"
	"os"
	"strings"
	"time"

	"Atom3/src/expr"
)

// Wait pauses execution for a given number of seconds
func Wait(seconds int) {
	time.Sleep(time.Duration(seconds) * time.Second)
}

func RunFile(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		errorf("[Error]: cannot open %s: %v\n", filename, err)
		return
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		errorf("[Error]: cannot read %s: %v\n", filename, err)
		return
	}

	execAll(CompileFile(filename, lines))
}

// Run executes an already compiled program.
func Run(program []Node) {
	execAll(program)
}

func RunSource(src string) {
	execAll(Compile(strings.Split(src, "\n")))
}

// Eval runs a single bracketed expression and returns its printable form. The
// repl uses it so typing [1 + 1] shows an answer instead of nothing.
func Eval(src string) (string, bool) {
	src = strings.TrimSpace(src)
	if !strings.HasPrefix(src, "[") || !strings.HasSuffix(src, "]") {
		return "", false
	}

	e, err := expr.Compile(src[1:len(src)-1], scope)
	if err != nil {
		return "", false
	}
	return e.Eval(scope).Display(), true
}

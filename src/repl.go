package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"Atom3/src/parser"
)

func repl() {
	fmt.Println("atom repl - ctrl+c to exit")

	in := bufio.NewScanner(os.Stdin)
	var pending []string
	depth := 0

	for {
		if depth > 0 {
			fmt.Print("... ")
		} else {
			fmt.Print(">>> ")
		}

		if !in.Scan() {
			fmt.Println()
			return
		}

		line := in.Text()
		trimmed := strings.TrimSpace(line)

		if depth == 0 && len(pending) == 0 {
			if out, ok := parser.Eval(trimmed); ok {
				fmt.Println(out)
				continue
			}
		}

		pending = append(pending, line)
		depth += strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
		if depth < 0 {
			depth = 0
		}

		if depth == 0 {
			parser.RunSource(strings.Join(pending, "\n"))
			pending = nil
		}
	}
}

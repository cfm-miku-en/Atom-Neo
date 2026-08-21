package main

import (
	"fmt"
	"os"
	"strings"

	"Atom3/src/bench"
	"Atom3/src/misc"
	"Atom3/src/parser"
	"Atom3/src/pkgs"
	"Atom3/src/stdlib"
)

func usage() {
	fmt.Println("Usage: atom run [file.atom]    run the project in this folder, or one file")
	fmt.Println("       atom run --watch       run and restart when files change")
	fmt.Println("       atom repl              try atom line by line")
	fmt.Println("       atom proxy --to ...    reverse proxy, optionally with https")
	fmt.Println("       atom fmt [file]        tidy indentation, --check to only report")
	fmt.Println("       atom install <pkg.zip>  install a package into atom_modules")
	fmt.Println("       atom benchmark [flags]  measure this build")
	fmt.Println("       atom <file.atom>        run one file")
	fmt.Println()
	fmt.Println("benchmark flags: --compare (against local python), --json, --ask <question>")
	fmt.Printf("questions: %s\n", strings.Join(bench.Questions, ", "))
}

func run(args []string) {
	for i, a := range args {
		if a == "--watch" || a == "-w" {
			watch(append(append([]string{}, args[:i]...), args[i+1:]...))
			return
		}
	}

	if len(args) > 0 {
		parser.RunFile(args[0])
		return
	}

	project, err := pkgs.LoadProject()
	if err != nil {
		fmt.Printf("[Error]: %v\n", err)
		return
	}
	parser.RunFile(project.Main)
}

func install(args []string) {
	if len(args) == 0 {
		usage()
		return
	}

	m, err := pkgs.Install(args[0])
	if err != nil {
		fmt.Printf("[Install Error]: %v\n", err)
		return
	}
	fmt.Printf("installed %s %s\n", m.Name, m.Version)
}

func benchmark(args []string) {
	compare := false
	asJSON := false
	question := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--compare", "-c":
			compare = true
		case "--json":
			asJSON = true
		case "--ask":
			if i+1 < len(args) {
				question = args[i+1]
				i++
			}
		}
	}

	if strings.EqualFold(question, "FasterThanPython") {
		compare = true
	}

	results := bench.Run(compare)

	switch {
	case question != "":
		bench.Ask(question, results)
	case asJSON:
		bench.JSON(results)
	default:
		bench.Table(results)
	}
}

// quiet keeps the banner out of output that is meant to be piped.
func quiet(args []string) bool {
	for _, a := range args {
		if a == "--json" {
			return true
		}
	}
	return false
}

func main() {
	args := os.Args[1:]

	if !quiet(args) {
		misc.ShowBanner()
	}
	stdlib.RegisterAll()

	if len(args) == 0 {
		usage()
		return
	}

	switch args[0] {
	case "run":
		run(args[1:])
	case "install":
		install(args[1:])
	case "benchmark":
		benchmark(args[1:])
	case "repl":
		repl()
	case "proxy":
		proxy(args[1:])
	case "fmt":
		format(args[1:])
	default:
		parser.RunFile(args[0])
	}
}

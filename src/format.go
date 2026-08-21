package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// formatSource re-indents by block depth and trims trailing space. It works on
// lines rather than a syntax tree because the compiler keeps no source
// positions, and because Atom statements are already one per line.
func formatSource(src string) string {
	src = strings.ReplaceAll(src, "\r\n", "\n")

	var out []string
	depth := 0
	blanks := 0

	for _, raw := range strings.Split(src, "\n") {
		line := strings.TrimSpace(raw)

		if line == "" {
			// Runs of blank lines collapse to one, and leading ones are dropped.
			blanks++
			continue
		}
		if blanks > 0 && len(out) > 0 {
			out = append(out, "")
		}
		blanks = 0

		if strings.HasPrefix(line, "}") && depth > 0 {
			depth--
		}

		out = append(out, strings.Repeat("\t", depth)+line)

		if strings.HasSuffix(line, "{") {
			depth++
		}
	}

	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, "\n") + "\n"
}

func atomFiles(root string) []string {
	var found []string

	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "atom_modules", "node_modules":
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".atom" {
			found = append(found, path)
		}
		return nil
	})
	return found
}

func format(args []string) {
	check := false
	var targets []string

	for _, a := range args {
		if a == "--check" {
			check = true
			continue
		}
		targets = append(targets, a)
	}

	if len(targets) == 0 {
		targets = atomFiles(".")
	}
	if len(targets) == 0 {
		fmt.Println("no .atom files here")
		return
	}

	changed := 0
	for _, path := range targets {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("[Format Error]: %v\n", err)
			continue
		}

		formatted := formatSource(string(data))
		if formatted == string(data) {
			continue
		}
		changed++

		if check {
			fmt.Println(path)
			continue
		}
		if err := os.WriteFile(path, []byte(formatted), 0o644); err != nil {
			fmt.Printf("[Format Error]: %v\n", err)
			continue
		}
		fmt.Println(path)
	}

	if check {
		if changed == 0 {
			fmt.Println("all formatted")
			return
		}
		fmt.Printf("%d file(s) need formatting\n", changed)
		os.Exit(1)
	}

	if changed == 0 {
		fmt.Println("all formatted")
	}
}

package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var watched = map[string]bool{
	".atom": true,
	".html": true,
	".css":  true,
	".js":   true,
	".json": true,
}

// fingerprint changes whenever a watched file is added, removed, resized or
// touched, which is cheaper and more portable than a filesystem event API.
func fingerprint() string {
	var b strings.Builder

	filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
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
		if !watched[filepath.Ext(path)] {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}
		fmt.Fprintf(&b, "%s:%d:%d;", path, info.ModTime().UnixNano(), info.Size())
		return nil
	})

	return b.String()
}

func watch(args []string) {
	self, err := os.Executable()
	if err != nil {
		fmt.Printf("[Error]: cannot locate the atom binary: %v\n", err)
		return
	}

	var child *exec.Cmd

	start := func() {
		child = exec.Command(self, append([]string{"run"}, args...)...)
		child.Stdout = os.Stdout
		child.Stderr = os.Stderr
		if err := child.Start(); err != nil {
			fmt.Printf("[Error]: %v\n", err)
		}
	}

	stop := func() {
		if child == nil || child.Process == nil {
			return
		}
		child.Process.Kill()
		child.Wait()
	}

	seen := fingerprint()
	fmt.Println("watching for changes, ctrl+c to stop")
	start()

	for {
		time.Sleep(500 * time.Millisecond)

		next := fingerprint()
		if next == seen {
			continue
		}
		seen = next

		fmt.Println("change detected, restarting")
		stop()
		start()
	}
}

package misc

import (
	"os"
	"os/exec"
	"runtime"

	"Atom3/src/expr"
)

// Version is stamped at build time with -ldflags "-X Atom3/src/misc.Version=...".
// A source build that skips that says dev, which is the honest answer.
var Version = "dev"

func ClearTerminal() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}
func ShowBanner() {
	ClearTerminal()

	println("Atom Neo")
	println("Release: " + Version)
	println("Engine: " + expr.Engine)
	println("A fork of Atom3 by WawaDev (formerly spacecat) and the Atom3 Team.")
	println("Three years of nothing, Third attempt at a new programming language and three mugs of tea.")
}

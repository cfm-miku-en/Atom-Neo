package parser

import (
	"os"
	"strconv"

	"Atom3/src/builtins"
)

// ExecuteFunction acts as the central router for built-in functions. Arguments
// arrive already evaluated.
func ExecuteFunction(name string, args []string) {
	switch name {
	case "print":
		for _, arg := range args {
			Print(arg)
		}

	case "exit":
		os.Exit(0)

	case "wait":
		for _, arg := range args {
			if num, err := strconv.Atoi(arg); err == nil {
				Wait(num)
			}
		}

	default:
		if fn, ok := builtins.Lookup(name); ok {
			fn(args)
			return
		}
		errorf("[Runtime Error]: Unknown function '%s'\n", name)
	}
}

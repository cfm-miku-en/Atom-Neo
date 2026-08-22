package parser

import "Atom3/src/expr"

var scope = expr.NewScope()

// Values are cleared but the name to slot mapping is kept, so anything compiled
// against this scope still points at the right places.
func ResetVariables() {
	scope.Clear()
}

func SetVariable(name string, value expr.Value) {
	scope.Set(name, value)
}

func GetVariable(name string) (expr.Value, bool) {
	return scope.Lookup(name)
}

// ResetAll clears every bit of interpreter state, so a test can run one program
// without the previous one's variables or functions leaking into it.
func ResetAll() {
	scope.Clear()
	userFuncs = make(map[string]*funcDef)
	imported = make(map[string]bool)
	flow = sigNone
	returnValue = expr.Value{}
	callDepth = 0
}

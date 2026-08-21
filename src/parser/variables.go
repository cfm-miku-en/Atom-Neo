package parser

import "Atom3/src/expr"

var scope = expr.NewScope()

func ResetVariables() {
	scope = expr.NewScope()
}

func SetVariable(name string, value expr.Value) {
	scope.Set(name, value)
}

func GetVariable(name string) (expr.Value, bool) {
	v, ok := scope.Vars[name]
	return v, ok
}

// ResetAll clears every bit of interpreter state, so a test can run one program
// without the previous one's variables or functions leaking into it.
func ResetAll() {
	scope = expr.NewScope()
	userFuncs = make(map[string]*funcDef)
	imported = make(map[string]bool)
	flow = sigNone
	returnValue = expr.Value{}
}

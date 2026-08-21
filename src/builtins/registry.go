package builtins

type Func func(args []string)

// ValueFunc is a module function that produces a result, so it can be used
// inside an expression rather than only for its side effect.
type ValueFunc func(args []string) string

type Module struct {
	Name   string
	Funcs  map[string]Func
	Values map[string]ValueFunc
}

var native = make(map[string]*Module)
var active = make(map[string]Func)
var activeValues = make(map[string]ValueFunc)

func RegisterModule(m *Module) {
	native[m.Name] = m
}

func Native(name string) (*Module, bool) {
	m, ok := native[name]
	return m, ok
}

// Activate exposes a module's functions under its namespace, so web.listen
// only becomes callable once the module has been imported.
func Activate(m *Module) {
	for name, fn := range m.Funcs {
		active[m.Name+"."+name] = fn
	}
	for name, fn := range m.Values {
		activeValues[m.Name+"."+name] = fn
	}
}

// Call runs a user-defined Atom function by name. The interpreter installs it
// so native modules can invoke handlers written in Atom.
var Call func(name string, args []string) (string, bool)

func Lookup(name string) (Func, bool) {
	fn, ok := active[name]
	return fn, ok
}

func LookupValue(name string) (ValueFunc, bool) {
	fn, ok := activeValues[name]
	return fn, ok
}

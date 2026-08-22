package parser

import (
	"sync"

	"Atom3/src/builtins"
	"Atom3/src/expr"
)

func init() {
	expr.Invoke = invoke
	expr.IsFunc = isUserFunc
	builtins.Call = callFromNative
}

func isUserFunc(name string) bool {
	_, ok := userFuncs[name]
	return ok
}

// The interpreter keeps its scope and control-flow state in package variables,
// so native callers are serialised rather than allowed to run Atom code on
// several goroutines at once.
var nativeCall sync.Mutex

func callFromNative(name string, args []string) (string, bool) {
	nativeCall.Lock()
	defer nativeCall.Unlock()

	vals := make([]expr.Value, len(args))
	for i, a := range args {
		vals[i] = expr.Str(a)
	}

	v, ok := invoke(name, vals)
	return v.Display(), ok
}

// Blocks are resolved into a tree once, before anything runs, so execution
// never re-parses a line or rediscovers where a block ends.
type Node interface {
	Exec()
}

type signal uint8

const (
	sigNone signal = iota
	sigReturn
	sigBreak
	sigContinue
)

var flow signal
var returnValue expr.Value

func execAll(nodes []Node) {
	for _, n := range nodes {
		n.Exec()
		if flow != sigNone {
			return
		}
	}
}

func truthy(v expr.Value) bool {
	return v.Kind == expr.Bool && v.Num != 0
}

// loopSignal reports whether the enclosing loop should stop.
func loopSignal() bool {
	switch flow {
	case sigBreak:
		flow = sigNone
		return true
	case sigContinue:
		flow = sigNone
	case sigReturn:
		return true
	}
	return false
}

type assignNode struct {
	name   string
	slot   int
	val    expr.Expr
	global bool
}

func (n *assignNode) Exec() {
	v := n.val.Eval(scope)
	if n.global {
		scope.SetSlot(n.slot, v)
		return
	}
	scope.SetVar(n.name, n.slot, v)
}

type ifNode struct {
	cond expr.Expr
	then []Node
	els  []Node
}

func (n *ifNode) Exec() {
	if truthy(n.cond.Eval(scope)) {
		execAll(n.then)
		return
	}
	execAll(n.els)
}

type whileNode struct {
	cond expr.Expr
	body []Node
}

func (n *whileNode) Exec() {
	for truthy(n.cond.Eval(scope)) {
		execAll(n.body)
		if loopSignal() {
			return
		}
	}
}

type repeatNode struct {
	count expr.Expr
	body  []Node
}

func (n *repeatNode) Exec() {
	count := int(n.count.Eval(scope).Num)
	for i := 0; i < count; i++ {
		execAll(n.body)
		if loopSignal() {
			return
		}
	}
}

type eachNode struct {
	name string
	list expr.Expr
	body []Node
}

func (n *eachNode) Exec() {
	subject := n.list.Eval(scope)

	items := subject.Elems()
	if subject.Kind == expr.Map {
		// Walking a map yields its keys, in sorted order so runs repeat.
		keys := subject.Keys()
		items = make([]expr.Value, len(keys))
		for i, k := range keys {
			items[i] = expr.Str(k)
		}
	}

	for _, item := range items {
		scope.Set(n.name, item)
		execAll(n.body)
		if loopSignal() {
			return
		}
	}
}

type returnNode struct{ val expr.Expr }

func (n *returnNode) Exec() {
	if n.val != nil {
		returnValue = n.val.Eval(scope)
	} else {
		returnValue = expr.Value{}
	}
	flow = sigReturn
}

type breakNode struct{}

func (breakNode) Exec() { flow = sigBreak }

type continueNode struct{}

func (continueNode) Exec() { flow = sigContinue }

type importNode struct{ name string }

func (n *importNode) Exec() { ImportModule(n.name) }

type funcDef struct {
	params []string
	body   []Node
}

// Definitions are gathered while compiling, so a function can be called from a
// line above the one that defines it.
var userFuncs = make(map[string]*funcDef)

func invoke(name string, args []expr.Value) (expr.Value, bool) {
	fn, ok := userFuncs[name]
	if !ok {
		if b, ok := valueBuiltins[name]; ok {
			return b(args), true
		}
		if b, ok := builtins.LookupValue(name); ok {
			strs := make([]string, len(args))
			for i, v := range args {
				strs[i] = v.Display()
			}
			return expr.Str(b(strs)), true
		}
		return expr.Value{}, false
	}

	if len(args) != len(fn.params) {
		errorf("[Runtime Error]: Function '%s' expects %d args, got %d\n", name, len(fn.params), len(args))
		return expr.Value{}, true
	}

	frame := make(map[string]expr.Value, len(fn.params))
	for i, p := range fn.params {
		frame[p] = args[i]
	}

	scope.Push(frame)
	execAll(fn.body)
	scope.Pop()

	result := returnValue
	returnValue = expr.Value{}
	flow = sigNone
	return result, true
}

type callNode struct {
	name string
	args []expr.Expr
}

func (n *callNode) Exec() {
	args := make([]expr.Value, len(n.args))
	for i, a := range n.args {
		args[i] = a.Eval(scope)
	}

	if _, ok := invoke(n.name, args); ok {
		return
	}

	strs := make([]string, len(args))
	for i, v := range args {
		strs[i] = v.Display()
	}
	ExecuteFunction(n.name, strs)
}

package expr

import "math"

type op uint8

const (
	opAdd op = iota
	opSub
	opMul
	opDiv
	opMod
	opEq
	opNe
	opLt
	opGt
	opLe
	opGe
	opAnd
	opOr
	opNot
	opNeg
	opIndex
)

// Scope holds the variables an expression can see. Globals live in numbered
// slots rather than a map: names are resolved to an index while compiling, so
// reading or writing one at run time is an array index instead of hashing a
// string. Each function call pushes a frame, so parameters do not leak and
// recursion keeps its own copies.
type Scope struct {
	slots  []Value
	filled []bool
	ids    map[string]int
	frames []map[string]Value
}

func NewScope() *Scope {
	return &Scope{ids: make(map[string]int)}
}

// Slot resolves a global name to its index, giving out a new one if the name is
// new. Call it while compiling, not while running.
func (s *Scope) Slot(name string) int {
	if id, ok := s.ids[name]; ok {
		return id
	}

	id := len(s.slots)
	s.ids[name] = id
	s.slots = append(s.slots, Value{})
	s.filled = append(s.filled, false)
	return id
}

// Clear empties every value but keeps the name to slot mapping, so expressions
// compiled earlier still point at the right places afterwards.
func (s *Scope) Clear() {
	for i := range s.slots {
		s.slots[i] = Value{}
		s.filled[i] = false
	}
	s.frames = nil
}

func (s *Scope) Push(frame map[string]Value) {
	s.frames = append(s.frames, frame)
}

func (s *Scope) Pop() {
	s.frames = s.frames[:len(s.frames)-1]
}

// Lookup reports whether a name has a value, which is what separates an unset
// global from one that genuinely holds zero.
func (s *Scope) Lookup(name string) (Value, bool) {
	if n := len(s.frames); n > 0 {
		if v, ok := s.frames[n-1][name]; ok {
			return v, true
		}
	}
	if id, ok := s.ids[name]; ok && s.filled[id] {
		return s.slots[id], true
	}
	return Value{}, false
}

func (s *Scope) Get(name string) Value {
	if v, ok := s.Lookup(name); ok {
		return v
	}

	// A bare name that is not a variable may still be a function, which is how
	// a handler gets passed to something like web.handle.
	if IsFunc != nil && IsFunc(name) {
		return FuncRef(name)
	}
	return Value{}
}

func (s *Scope) Set(name string, v Value) {
	if n := len(s.frames); n > 0 {
		s.frames[n-1][name] = v
		return
	}
	s.SetGlobal(name, v)
}

// SetGlobal writes past any active call frame, which is what the global
// keyword compiles to.
func (s *Scope) SetGlobal(name string, v Value) {
	s.SetSlot(s.Slot(name), v)
}

// SetSlot writes straight to a global slot, ignoring any call frame.
func (s *Scope) SetSlot(id int, v Value) {
	s.slots[id] = v
	s.filled[id] = true
}

// SetVar writes to the innermost call frame when there is one and to the given
// global slot otherwise, which is what a plain assignment does.
func (s *Scope) SetVar(name string, slot int, v Value) {
	if n := len(s.frames); n > 0 {
		s.frames[n-1][name] = v
		return
	}
	s.SetSlot(slot, v)
}

// Invoke is installed by the interpreter so expressions can call user-defined
// functions without this package depending on the interpreter.
var Invoke func(name string, args []Value) (Value, bool)

// IsFunc reports whether a name refers to a user-defined function.
var IsFunc func(name string) bool

type callNode struct {
	name string
	args []Expr
}

func (n callNode) Eval(s *Scope) Value {
	if Invoke == nil {
		return Value{}
	}

	args := make([]Value, len(n.args))
	for i, a := range n.args {
		args[i] = a.Eval(s)
	}

	v, _ := Invoke(n.name, args)
	return v
}

type Expr interface {
	Eval(s *Scope) Value
}

type literalNode struct{ v Value }

func (n literalNode) Eval(*Scope) Value { return n.v }

type listNode struct{ items []Expr }

func (n listNode) Eval(s *Scope) Value {
	vals := make([]Value, len(n.items))
	for i, e := range n.items {
		vals[i] = e.Eval(s)
	}
	return NewList(vals)
}

type mapPair struct {
	key string
	val Expr
}

type mapNode struct{ pairs []mapPair }

func (n mapNode) Eval(s *Scope) Value {
	dict := make(map[string]Value, len(n.pairs))
	for _, p := range n.pairs {
		dict[p.key] = p.val.Eval(s)
	}
	return NewMap(dict)
}

type varNode struct {
	name string
	slot int
}

func (n *varNode) Eval(s *Scope) Value {
	// Only a function call creates a frame, so the common case skips this.
	if fn := len(s.frames); fn > 0 {
		if v, ok := s.frames[fn-1][n.name]; ok {
			return v
		}
	}
	if n.slot < len(s.filled) && s.filled[n.slot] {
		return s.slots[n.slot]
	}
	return s.Get(n.name)
}

type unaryNode struct {
	op op
	x  Expr
}

func (n unaryNode) Eval(s *Scope) Value {
	v := n.x.Eval(s)
	if n.op == opNeg {
		return Num(-v.Num)
	}
	return Boolean(!v.Truthy())
}

type binaryNode struct {
	op   op
	l, r Expr
}

func equal(l, r Value) bool {
	// Numbers are the overwhelmingly common case, so they answer before the
	// collection checks rather than after them.
	if l.Kind == Number && r.Kind == Number {
		return l.Num == r.Num
	}
	if l.Kind == Map || r.Kind == Map {
		if l.Kind != r.Kind {
			return false
		}
		a, b := l.Dict(), r.Dict()
		if len(a) != len(b) {
			return false
		}
		for k, v := range a {
			other, ok := b[k]
			if !ok || !equal(v, other) {
				return false
			}
		}
		return true
	}
	if l.Kind == List || r.Kind == List {
		a, b := l.Elems(), r.Elems()
		if l.Kind != r.Kind || len(a) != len(b) {
			return false
		}
		for i := range a {
			if !equal(a[i], b[i]) {
				return false
			}
		}
		return true
	}
	if l.Kind == Text || r.Kind == Text {
		return l.Display() == r.Display()
	}
	return l.Num == r.Num
}

func (n binaryNode) Eval(s *Scope) Value {
	switch n.op {
	case opAnd:
		if !n.l.Eval(s).Truthy() {
			return Boolean(false)
		}
		return Boolean(n.r.Eval(s).Truthy())
	case opOr:
		if n.l.Eval(s).Truthy() {
			return Boolean(true)
		}
		return Boolean(n.r.Eval(s).Truthy())
	}

	l := n.l.Eval(s)
	r := n.r.Eval(s)

	switch n.op {
	case opIndex:
		if l.Kind == Map {
			return l.Dict()[r.Display()]
		}
		items := l.Elems()
		i := int(r.Num)
		if i < 0 || i >= len(items) {
			return Value{}
		}
		return items[i]
	case opAdd:
		if l.Kind == Number && r.Kind == Number {
			return Num(l.Num + r.Num)
		}
		if l.Kind == List && r.Kind == List {
			joined := append(append([]Value{}, l.Elems()...), r.Elems()...)
			return NewList(joined)
		}
		if l.Kind == Text || r.Kind == Text {
			return Str(l.Display() + r.Display())
		}
		return Num(l.Num + r.Num)
	case opSub:
		return Num(l.Num - r.Num)
	case opMul:
		return Num(l.Num * r.Num)
	case opDiv:
		return Num(l.Num / r.Num)
	case opMod:
		return Num(math.Mod(l.Num, r.Num))
	case opEq:
		return Boolean(equal(l, r))
	case opNe:
		return Boolean(!equal(l, r))
	case opLt:
		return Boolean(l.Num < r.Num)
	case opGt:
		return Boolean(l.Num > r.Num)
	case opLe:
		return Boolean(l.Num <= r.Num)
	case opGe:
		return Boolean(l.Num >= r.Num)
	}
	return Value{}
}

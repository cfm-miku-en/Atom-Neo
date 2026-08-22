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

// Ref says where a name lives. Working that out while compiling turns every
// read and write at run time into an array index instead of hashing a string.
type Ref struct {
	Local bool
	Index int
	Name  string
}

// Scope holds the variables an expression can see.
//
// Globals sit in numbered slots and function locals sit in a shared stack, both
// addressed by index. Calls used to build a map per invocation, which was the
// overwhelming majority of everything the interpreter allocated.
type Scope struct {
	slots []Value
	ids   map[string]int

	// Runtime frame storage; base is where the running call's locals start.
	stack []Value
	base  int

	// Compile time only: the local table of the function being compiled, and
	// the enclosing ones, since a func can be written inside another.
	locals    map[string]int
	enclosing []map[string]int
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
	s.slots = append(s.slots, Value{Kind: Unset})
	return id
}

// Define records a name being assigned: a local inside a function, a global at
// the top level.
func (s *Scope) Define(name string) Ref {
	if s.locals == nil {
		return Ref{Index: s.Slot(name), Name: name}
	}

	i, ok := s.locals[name]
	if !ok {
		i = len(s.locals)
		s.locals[name] = i
	}
	return Ref{Local: true, Index: i, Name: name}
}

// Resolve records a name being read. A name that is not a local of the function
// being compiled refers to a global.
func (s *Scope) Resolve(name string) Ref {
	if s.locals != nil {
		if i, ok := s.locals[name]; ok {
			return Ref{Local: true, Index: i, Name: name}
		}
	}
	return Ref{Index: s.Slot(name), Name: name}
}

// BeginFunc opens a local table with the parameters already in it. EndFunc
// closes it and reports how many stack entries a call to it needs.
func (s *Scope) BeginFunc(params []string) {
	s.enclosing = append(s.enclosing, s.locals)

	s.locals = make(map[string]int, len(params)+4)
	for i, p := range params {
		s.locals[p] = i
	}
}

func (s *Scope) EndFunc() int {
	count := len(s.locals)

	s.locals = s.enclosing[len(s.enclosing)-1]
	s.enclosing = s.enclosing[:len(s.enclosing)-1]
	return count
}

// PushFrame reserves room for a call's locals and returns the previous base for
// PopFrame to restore. The stack is reused between calls, so once it has grown
// enough this stops allocating.
func (s *Scope) PushFrame(count int) int {
	base := len(s.stack)
	for i := 0; i < count; i++ {
		s.stack = append(s.stack, Value{})
	}

	prev := s.base
	s.base = base
	return prev
}

func (s *Scope) PopFrame(prev int) {
	s.stack = s.stack[:s.base]
	s.base = prev
}

func (s *Scope) SetLocal(i int, v Value) { s.stack[s.base+i] = v }

// Store writes through a ref, which is what an assignment compiles to.
func (s *Scope) Store(ref Ref, v Value) {
	if ref.Local {
		s.stack[s.base+ref.Index] = v
		return
	}
	s.SetSlot(ref.Index, v)
}

// Clear empties every value but keeps the name to slot mapping, so expressions
// compiled earlier still point at the right places afterwards.
func (s *Scope) Clear() {
	for i := range s.slots {
		s.slots[i] = Value{Kind: Unset}
	}
	s.stack = s.stack[:0]
	s.base = 0
}

// Lookup finds a global by name. Locals are addressed by index and carry no
// names at run time, so they are not reachable this way.
func (s *Scope) Lookup(name string) (Value, bool) {
	if id, ok := s.ids[name]; ok && s.slots[id].Kind != Unset {
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
}

// Invoke is installed by the interpreter so expressions can call user-defined
// functions without this package depending on the interpreter.
var Invoke func(name string, args []Value) (Value, bool)

// IsFunc reports whether a name refers to a user-defined function.
var IsFunc func(name string) bool

// Unknown is called when an expression tries to call something that does not
// exist. The interpreter installs it; without one the call quietly produces an
// empty value, which is how a misspelled name in a negated test used to let a
// branch run.
var Unknown func(name string)

// Expr is a compiled expression.
//
// It is a function rather than an interface because Go dispatches a closure
// several times faster than an interface method, and because building one
// per operator lets the operator be chosen while compiling instead of being
// switched on during every visit.
type Expr func(s *Scope) Value

func literal(v Value) Expr {
	return func(*Scope) Value { return v }
}

func variable(ref Ref) Expr {
	if ref.Local {
		return func(s *Scope) Value { return s.stack[s.base+ref.Index] }
	}

	return func(s *Scope) Value {
		if v := s.slots[ref.Index]; v.Kind != Unset {
			return v
		}

		// A bare name that is not a variable may still be a function, which is
		// how a handler gets passed to something like web.handle.
		if IsFunc != nil && IsFunc(ref.Name) {
			return FuncRef(ref.Name)
		}
		return Value{}
	}
}

func list(items []Expr) Expr {
	return func(s *Scope) Value {
		vals := make([]Value, len(items))
		for i, item := range items {
			vals[i] = item(s)
		}
		return NewList(vals)
	}
}

type mapPair struct {
	key string
	val Expr
}

func dict(pairs []mapPair) Expr {
	return func(s *Scope) Value {
		d := make(map[string]Value, len(pairs))
		for _, p := range pairs {
			d[p.key] = p.val(s)
		}
		return NewMap(d)
	}
}

func call(name string, args []Expr) Expr {
	return func(s *Scope) Value {
		if Invoke == nil {
			return Value{}
		}

		// Arguments are evaluated onto the shared stack and handed over as a
		// view of it, so an ordinary call allocates nothing. Anything reached
		// through Invoke must copy what it wants to keep rather than holding
		// on to the slice.
		base := len(s.stack)
		for _, a := range args {
			s.stack = append(s.stack, a(s))
		}

		v, ok := Invoke(name, s.stack[base:])
		s.stack = s.stack[:base]

		if !ok && Unknown != nil {
			Unknown(name)
		}
		return v
	}
}

func unary(o op, x Expr) Expr {
	if o == opNeg {
		return func(s *Scope) Value { return Num(-x(s).Num) }
	}
	return func(s *Scope) Value { return Boolean(!x(s).Truthy()) }
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

// join handles the cases of + that are not two numbers.
func join(a, b Value) Value {
	if a.Kind == List && b.Kind == List {
		return NewList(append(append([]Value{}, a.Elems()...), b.Elems()...))
	}
	if a.Kind == Text || b.Kind == Text {
		return Str(a.Display() + b.Display())
	}
	return Num(a.Num + b.Num)
}

func element(container, key Value) Value {
	if container.Kind == Map {
		return container.Dict()[key.Display()]
	}

	items := container.Elems()
	i := int(key.Num)
	if i < 0 || i >= len(items) {
		return Value{}
	}
	return items[i]
}

// binary picks the operation once, while compiling. Evaluating one is then a
// closure call rather than a switch taken on every visit.
func binary(o op, l, r Expr) Expr {
	switch o {
	case opAnd:
		return func(s *Scope) Value {
			if !l(s).Truthy() {
				return Boolean(false)
			}
			return Boolean(r(s).Truthy())
		}
	case opOr:
		return func(s *Scope) Value {
			if l(s).Truthy() {
				return Boolean(true)
			}
			return Boolean(r(s).Truthy())
		}
	case opAdd:
		return func(s *Scope) Value {
			a, b := l(s), r(s)
			if a.Kind == Number && b.Kind == Number {
				return Num(a.Num + b.Num)
			}
			return join(a, b)
		}
	case opSub:
		return func(s *Scope) Value { return Num(l(s).Num - r(s).Num) }
	case opMul:
		return func(s *Scope) Value { return Num(l(s).Num * r(s).Num) }
	case opDiv:
		return func(s *Scope) Value { return Num(l(s).Num / r(s).Num) }
	case opMod:
		return func(s *Scope) Value { return Num(math.Mod(l(s).Num, r(s).Num)) }
	case opEq:
		return func(s *Scope) Value { return Boolean(equal(l(s), r(s))) }
	case opNe:
		return func(s *Scope) Value { return Boolean(!equal(l(s), r(s))) }
	case opLt:
		return func(s *Scope) Value {
			if l(s).Num < r(s).Num {
				return yes
			}
			return no
		}
	case opGt:
		return func(s *Scope) Value {
			if l(s).Num > r(s).Num {
				return yes
			}
			return no
		}
	case opLe:
		return func(s *Scope) Value {
			if l(s).Num <= r(s).Num {
				return yes
			}
			return no
		}
	case opGe:
		return func(s *Scope) Value {
			if l(s).Num >= r(s).Num {
				return yes
			}
			return no
		}
	case opIndex:
		return func(s *Scope) Value { return element(l(s), r(s)) }
	}

	return func(*Scope) Value { return Value{} }
}

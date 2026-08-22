package expr

import "testing"

// Which dispatch mechanism Go executes fastest decides whether the evaluator
// should stay an interface tree or become a tree of closures. Measured here
// rather than taken on faith from an article.

type ifaceNode interface{ eval(*Scope) float64 }

type ifaceLit struct{ v float64 }

func (n ifaceLit) eval(*Scope) float64 { return n.v }

type ifaceAdd struct{ l, r ifaceNode }

func (n ifaceAdd) eval(s *Scope) float64 { return n.l.eval(s) + n.r.eval(s) }

type ifaceVar struct{ slot int }

func (n ifaceVar) eval(s *Scope) float64 { return s.slots[n.slot].Num }

type closure func(*Scope) float64

func closureLit(v float64) closure { return func(*Scope) float64 { return v } }

func closureVar(slot int) closure {
	return func(s *Scope) float64 { return s.slots[slot].Num }
}

func closureAdd(l, r closure) closure {
	return func(s *Scope) float64 { return l(s) + r(s) }
}

func benchScope() *Scope {
	s := NewScope()
	s.SetGlobal("x", Num(3))
	s.SetGlobal("y", Num(4))
	return s
}

// (x + y) + (x + 1), the shape of a real expression rather than a single op.
func BenchmarkDispatchInterface(b *testing.B) {
	s := benchScope()
	tree := ifaceNode(ifaceAdd{
		l: ifaceAdd{l: ifaceVar{0}, r: ifaceVar{1}},
		r: ifaceAdd{l: ifaceVar{0}, r: ifaceLit{1}},
	})

	b.ReportAllocs()
	b.ResetTimer()

	var total float64
	for i := 0; i < b.N; i++ {
		total += tree.eval(s)
	}
	_ = total
}

func BenchmarkDispatchClosure(b *testing.B) {
	s := benchScope()
	tree := closureAdd(
		closureAdd(closureVar(0), closureVar(1)),
		closureAdd(closureVar(0), closureLit(1)),
	)

	b.ReportAllocs()
	b.ResetTimer()

	var total float64
	for i := 0; i < b.N; i++ {
		total += tree(s)
	}
	_ = total
}

// The same shape through the real evaluator, so the two microbenchmarks above
// have something to be compared against.
func BenchmarkDispatchReal(b *testing.B) {
	s := benchScope()

	e, err := Compile("(x + y) + (x + 1)", s)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		e(s)
	}
}

// A bytecode virtual machine is the usual next step after a tree walker, so it
// is measured here too rather than argued about. The program is the same
// expression: (x + y) + (x + 1).
type instr struct {
	code byte
	arg  float64
	slot int
}

const (
	opcPush byte = iota
	opcLoad
	opcPlus
)

func BenchmarkDispatchBytecode(b *testing.B) {
	s := benchScope()

	program := []instr{
		{code: opcLoad, slot: 0},
		{code: opcLoad, slot: 1},
		{code: opcPlus},
		{code: opcLoad, slot: 0},
		{code: opcPush, arg: 1},
		{code: opcPlus},
		{code: opcPlus},
	}

	stack := make([]float64, 0, 8)

	b.ReportAllocs()
	b.ResetTimer()

	var total float64
	for i := 0; i < b.N; i++ {
		stack = stack[:0]

		// Go has no computed goto, so every opcode leaves through this one
		// switch. In C each opcode would end with its own indirect jump.
		for pc := 0; pc < len(program); pc++ {
			in := program[pc]

			switch in.code {
			case opcPush:
				stack = append(stack, in.arg)
			case opcLoad:
				stack = append(stack, s.slots[in.slot].Num)
			case opcPlus:
				n := len(stack)
				stack[n-2] += stack[n-1]
				stack = stack[:n-1]
			}
		}
		total += stack[0]
	}
	_ = total
}

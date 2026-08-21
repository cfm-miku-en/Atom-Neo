package expr

import (
	"sort"
	"strconv"
	"strings"
)

// Engine is the name of the evaluator Atom runs on.
const Engine = "Quark"

type Kind uint8

const (
	Number Kind = iota
	Text
	Bool
	List
	Map
	Func
)

// box carries whichever collection a Value holds. A value is never both a list
// and a map, so one pointer serves both and Value stays four words wide; at
// five words Go stops returning it in registers, which measured as a two thirds
// loss of interpreter speed.
type box struct {
	list []Value
	dict map[string]Value
}

type Value struct {
	Kind Kind
	Num  float64
	str  *string
	box  *box
}

func Num(f float64) Value { return Value{Kind: Number, Num: f} }
func Str(s string) Value  { return Value{Kind: Text, str: &s} }

func Boolean(b bool) Value {
	v := Value{Kind: Bool}
	if b {
		v.Num = 1
	}
	return v
}

// FuncRef names a user-defined function so it can be passed around as a value.
func FuncRef(name string) Value {
	return Value{Kind: Func, str: &name}
}

func NewList(items []Value) Value {
	return Value{Kind: List, box: &box{list: items}}
}

func NewMap(dict map[string]Value) Value {
	return Value{Kind: Map, box: &box{dict: dict}}
}

func (v Value) Text() string {
	if v.str == nil {
		return ""
	}
	return *v.str
}

func (v Value) Elems() []Value {
	if v.Kind != List || v.box == nil {
		return nil
	}
	return v.box.list
}

func (v Value) Dict() map[string]Value {
	if v.Kind != Map || v.box == nil {
		return nil
	}
	return v.box.dict
}

// Append grows a list in place so push shares the change with every holder.
func (v Value) Append(items ...Value) {
	if v.Kind == List && v.box != nil {
		v.box.list = append(v.box.list, items...)
	}
}

func (v Value) Truncate(n int) {
	if v.Kind == List && v.box != nil {
		v.box.list = v.box.list[:n]
	}
}

func (v Value) Put(key string, item Value) {
	if v.Kind == Map && v.box != nil {
		if v.box.dict == nil {
			v.box.dict = make(map[string]Value)
		}
		v.box.dict[key] = item
	}
}

func (v Value) Delete(key string) {
	if v.Kind == Map && v.box != nil {
		delete(v.box.dict, key)
	}
}

// Keys are sorted so printing a map and iterating one are both repeatable.
func (v Value) Keys() []string {
	dict := v.Dict()
	keys := make([]string, 0, len(dict))
	for k := range dict {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (v Value) Truthy() bool {
	switch v.Kind {
	case Text, Func:
		return v.Text() != ""
	case List:
		return len(v.Elems()) > 0
	case Map:
		return len(v.Dict()) > 0
	default:
		return v.Num != 0
	}
}

func (v Value) Display() string {
	switch v.Kind {
	case Text, Func:
		return v.Text()
	case Bool:
		if v.Num != 0 {
			return "true"
		}
		return "false"
	case List:
		items := v.Elems()
		parts := make([]string, len(items))
		for i, it := range items {
			parts[i] = it.Display()
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case Map:
		dict := v.Dict()
		keys := v.Keys()
		parts := make([]string, len(keys))
		for i, k := range keys {
			parts[i] = k + ": " + dict[k].Display()
		}
		return "[" + strings.Join(parts, ", ") + "]"
	default:
		return strconv.FormatFloat(v.Num, 'f', -1, 64)
	}
}

// Literal reads a bare source token such as 42, "hi" or true into a Value.
func Literal(s string) Value {
	s = strings.TrimSpace(s)
	switch s {
	case "true":
		return Boolean(true)
	case "false":
		return Boolean(false)
	}
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return Str(Unescape(s[1 : len(s)-1]))
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return Num(f)
	}
	return Str(strings.Trim(s, `"`))
}

package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"Atom3/src/expr"
)

// One scanner for the whole run so successive input() calls continue through
// stdin rather than each starting over.
var stdin = bufio.NewScanner(os.Stdin)

type valueFunc func(args []expr.Value) expr.Value

// These take and return Values rather than strings so lists survive a call.
var valueBuiltins = map[string]valueFunc{
	"len": func(a []expr.Value) expr.Value {
		if len(a) != 1 {
			return expr.Value{}
		}
		switch a[0].Kind {
		case expr.List:
			return expr.Num(float64(len(a[0].Elems())))
		case expr.Map:
			return expr.Num(float64(len(a[0].Dict())))
		}
		return expr.Num(float64(len(a[0].Display())))
	},

	"push": func(a []expr.Value) expr.Value {
		if len(a) < 2 || a[0].Kind != expr.List {
			return expr.Value{}
		}
		a[0].Append(a[1:]...)
		return a[0]
	},

	"pop": func(a []expr.Value) expr.Value {
		if len(a) != 1 || a[0].Kind != expr.List {
			return expr.Value{}
		}
		items := a[0].Elems()
		if len(items) == 0 {
			return expr.Value{}
		}
		a[0].Truncate(len(items) - 1)
		return items[len(items)-1]
	},

	"first": func(a []expr.Value) expr.Value {
		if len(a) != 1 {
			return expr.Value{}
		}
		if items := a[0].Elems(); len(items) > 0 {
			return items[0]
		}
		return expr.Value{}
	},

	"last": func(a []expr.Value) expr.Value {
		if len(a) != 1 {
			return expr.Value{}
		}
		if items := a[0].Elems(); len(items) > 0 {
			return items[len(items)-1]
		}
		return expr.Value{}
	},

	"reverse": func(a []expr.Value) expr.Value {
		if len(a) != 1 {
			return expr.Value{}
		}
		items := a[0].Elems()
		out := make([]expr.Value, len(items))
		for i, v := range items {
			out[len(items)-1-i] = v
		}
		return expr.NewList(out)
	},

	"range": func(a []expr.Value) expr.Value {
		start, stop := 0.0, 0.0
		switch len(a) {
		case 1:
			stop = a[0].Num
		case 2:
			start, stop = a[0].Num, a[1].Num
		default:
			return expr.Value{}
		}
		var out []expr.Value
		for i := start; i < stop; i++ {
			out = append(out, expr.Num(i))
		}
		return expr.NewList(out)
	},

	"contains": func(a []expr.Value) expr.Value {
		if len(a) != 2 {
			return expr.Boolean(false)
		}
		if a[0].Kind == expr.List {
			for _, v := range a[0].Elems() {
				if v.Display() == a[1].Display() {
					return expr.Boolean(true)
				}
			}
			return expr.Boolean(false)
		}
		return expr.Boolean(strings.Contains(a[0].Display(), a[1].Display()))
	},

	// Anything from a request that ends up in a page has to go through this,
	// or a query parameter becomes a script tag.
	"escape": func(a []expr.Value) expr.Value {
		if len(a) != 1 {
			return expr.Str("")
		}
		return expr.Str(html.EscapeString(a[0].Display()))
	},

	"upper": func(a []expr.Value) expr.Value {
		if len(a) != 1 {
			return expr.Value{}
		}
		return expr.Str(strings.ToUpper(a[0].Display()))
	},

	"lower": func(a []expr.Value) expr.Value {
		if len(a) != 1 {
			return expr.Value{}
		}
		return expr.Str(strings.ToLower(a[0].Display()))
	},

	"trim": func(a []expr.Value) expr.Value {
		if len(a) != 1 {
			return expr.Value{}
		}
		return expr.Str(strings.TrimSpace(a[0].Display()))
	},

	"split": func(a []expr.Value) expr.Value {
		if len(a) != 2 {
			return expr.Value{}
		}
		parts := strings.Split(a[0].Display(), a[1].Display())
		out := make([]expr.Value, len(parts))
		for i, p := range parts {
			out[i] = expr.Str(p)
		}
		return expr.NewList(out)
	},

	"join": func(a []expr.Value) expr.Value {
		if len(a) != 2 {
			return expr.Value{}
		}
		items := a[0].Elems()
		parts := make([]string, len(items))
		for i, v := range items {
			parts[i] = v.Display()
		}
		return expr.Str(strings.Join(parts, a[1].Display()))
	},

	"floor": func(a []expr.Value) expr.Value { return math1(a, math.Floor) },
	"ceil":  func(a []expr.Value) expr.Value { return math1(a, math.Ceil) },
	"abs":   func(a []expr.Value) expr.Value { return math1(a, math.Abs) },
	"sqrt":  func(a []expr.Value) expr.Value { return math1(a, math.Sqrt) },
	"round": func(a []expr.Value) expr.Value { return math1(a, math.Round) },

	"min": func(a []expr.Value) expr.Value {
		if len(a) != 2 {
			return expr.Value{}
		}
		return expr.Num(math.Min(a[0].Num, a[1].Num))
	},

	"max": func(a []expr.Value) expr.Value {
		if len(a) != 2 {
			return expr.Value{}
		}
		return expr.Num(math.Max(a[0].Num, a[1].Num))
	},

	"random": func(a []expr.Value) expr.Value {
		return expr.Num(rand.Float64())
	},

	"randint": func(a []expr.Value) expr.Value {
		if len(a) != 2 {
			return expr.Value{}
		}
		lo, hi := int(a[0].Num), int(a[1].Num)
		if hi <= lo {
			return expr.Num(float64(lo))
		}
		return expr.Num(float64(lo + rand.Intn(hi-lo+1)))
	},

	"str": func(a []expr.Value) expr.Value {
		if len(a) != 1 {
			return expr.Value{}
		}
		return expr.Str(a[0].Display())
	},

	"num": func(a []expr.Value) expr.Value {
		if len(a) != 1 {
			return expr.Value{}
		}
		f, err := strconv.ParseFloat(strings.TrimSpace(a[0].Display()), 64)
		if err != nil {
			return expr.Num(0)
		}
		return expr.Num(f)
	},

	"read": func(a []expr.Value) expr.Value {
		if len(a) != 1 {
			return expr.Str("")
		}
		data, err := os.ReadFile(a[0].Display())
		if err != nil {
			return expr.Str("")
		}
		return expr.Str(string(data))
	},

	"write": func(a []expr.Value) expr.Value {
		if len(a) != 2 {
			return expr.Boolean(false)
		}
		return expr.Boolean(os.WriteFile(a[0].Display(), []byte(a[1].Display()), 0o644) == nil)
	},

	"append": func(a []expr.Value) expr.Value {
		if len(a) != 2 {
			return expr.Boolean(false)
		}
		f, err := os.OpenFile(a[0].Display(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return expr.Boolean(false)
		}
		defer f.Close()

		_, err = f.WriteString(a[1].Display())
		return expr.Boolean(err == nil)
	},

	"exists": func(a []expr.Value) expr.Value {
		if len(a) != 1 {
			return expr.Boolean(false)
		}
		_, err := os.Stat(a[0].Display())
		return expr.Boolean(err == nil)
	},

	"remove": func(a []expr.Value) expr.Value {
		if len(a) != 1 {
			return expr.Boolean(false)
		}
		return expr.Boolean(os.Remove(a[0].Display()) == nil)
	},

	// Strings carry no escape sequences, so splitting on a newline needs its
	// own helper rather than split(text, "\n").
	"lines": func(a []expr.Value) expr.Value {
		if len(a) != 1 {
			return expr.Value{}
		}
		text := strings.ReplaceAll(a[0].Display(), "\r\n", "\n")
		parts := strings.Split(strings.TrimRight(text, "\n"), "\n")

		out := make([]expr.Value, len(parts))
		for i, p := range parts {
			out[i] = expr.Str(p)
		}
		return expr.NewList(out)
	},

	"dict": func(a []expr.Value) expr.Value {
		return expr.NewMap(make(map[string]expr.Value))
	},

	"keys": func(a []expr.Value) expr.Value {
		if len(a) != 1 {
			return expr.Value{}
		}
		names := a[0].Keys()
		out := make([]expr.Value, len(names))
		for i, k := range names {
			out[i] = expr.Str(k)
		}
		return expr.NewList(out)
	},

	"values": func(a []expr.Value) expr.Value {
		if len(a) != 1 {
			return expr.Value{}
		}
		dict := a[0].Dict()
		names := a[0].Keys()
		out := make([]expr.Value, len(names))
		for i, k := range names {
			out[i] = dict[k]
		}
		return expr.NewList(out)
	},

	"has": func(a []expr.Value) expr.Value {
		if len(a) != 2 {
			return expr.Boolean(false)
		}
		_, ok := a[0].Dict()[a[1].Display()]
		return expr.Boolean(ok)
	},

	"put": func(a []expr.Value) expr.Value {
		if len(a) != 3 {
			return expr.Value{}
		}
		a[0].Put(a[1].Display(), a[2])
		return a[0]
	},

	"del": func(a []expr.Value) expr.Value {
		if len(a) != 2 {
			return expr.Value{}
		}
		a[0].Delete(a[1].Display())
		return a[0]
	},

	"fromjson": func(a []expr.Value) expr.Value {
		if len(a) != 1 {
			return expr.Value{}
		}
		var parsed any
		if err := json.Unmarshal([]byte(a[0].Display()), &parsed); err != nil {
			return expr.Value{}
		}
		return fromJSON(parsed)
	},

	"tojson": func(a []expr.Value) expr.Value {
		if len(a) != 1 {
			return expr.Str("")
		}
		data, err := json.Marshal(toJSON(a[0]))
		if err != nil {
			return expr.Str("")
		}
		return expr.Str(string(data))
	},

	"input": func(a []expr.Value) expr.Value {
		if len(a) == 1 {
			fmt.Fprint(Output, a[0].Display())
		}
		if !stdin.Scan() {
			return expr.Str("")
		}
		return expr.Str(stdin.Text())
	},
}

func math1(a []expr.Value, fn func(float64) float64) expr.Value {
	if len(a) != 1 {
		return expr.Value{}
	}
	return expr.Num(fn(a[0].Num))
}

func fromJSON(v any) expr.Value {
	switch t := v.(type) {
	case bool:
		return expr.Boolean(t)
	case float64:
		return expr.Num(t)
	case string:
		return expr.Str(t)
	case []any:
		out := make([]expr.Value, len(t))
		for i, e := range t {
			out[i] = fromJSON(e)
		}
		return expr.NewList(out)
	case map[string]any:
		dict := make(map[string]expr.Value, len(t))
		for k, e := range t {
			dict[k] = fromJSON(e)
		}
		return expr.NewMap(dict)
	}
	return expr.Str("")
}

func toJSON(v expr.Value) any {
	switch v.Kind {
	case expr.Number:
		return v.Num
	case expr.Bool:
		return v.Num != 0
	case expr.List:
		items := v.Elems()
		out := make([]any, len(items))
		for i, e := range items {
			out[i] = toJSON(e)
		}
		return out
	case expr.Map:
		dict := v.Dict()
		out := make(map[string]any, len(dict))
		for k, e := range dict {
			out[k] = toJSON(e)
		}
		return out
	}
	return v.Display()
}

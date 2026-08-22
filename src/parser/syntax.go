package parser

import (
	"fmt"
	"strings"

	"Atom3/src/expr"
)

type compiler struct {
	lines []string
	pos   int
	file  string
}

func Compile(lines []string) []Node {
	return CompileFile("", lines)
}

func CompileFile(file string, lines []string) []Node {
	c := &compiler{lines: lines, file: file}
	return c.block(false)
}

func (c *compiler) fail(at int, line, format string, args ...any) {
	where := fmt.Sprintf("line %d", at+1)
	if c.file != "" {
		where = fmt.Sprintf("%s line %d", c.file, at+1)
	}
	errorf("[Parse Error] %s: %s\n    %s\n", where, fmt.Sprintf(format, args...), strings.TrimSpace(line))
}

func skippable(line string) bool {
	return line == "" || strings.HasPrefix(line, "//")
}

// firstWord is the leading keyword of a line, stopping at whatever punctuation
// follows it so both "if [x] {" and "if[x] {" read as if.
func firstWord(line string) string {
	if i := strings.IndexAny(line, " \t([{"); i >= 0 {
		return line[:i]
	}
	return line
}

func (c *compiler) block(nested bool) []Node {
	var nodes []Node

	for c.pos < len(c.lines) {
		line := strings.TrimSpace(c.lines[c.pos])

		if skippable(line) {
			c.pos++
			continue
		}
		if line == "}" {
			c.pos++
			if nested {
				return nodes
			}
			continue
		}

		if n := c.statement(line); n != nil {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

// nextMeaningful reports the index of the next line that is not blank or a
// comment, so an else separated from its brace by whitespace still attaches.
func (c *compiler) nextMeaningful() int {
	i := c.pos
	for i < len(c.lines) && skippable(strings.TrimSpace(c.lines[i])) {
		i++
	}
	return i
}

func conditionOf(line string) (expr.Expr, error) {
	start := strings.Index(line, "[")
	end := strings.LastIndex(line, "]")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("missing a [condition]")
	}
	return expr.Compile(line[start+1:end], scope)
}

func compileValue(tok string) (expr.Expr, error) {
	if strings.HasPrefix(tok, "[") && strings.HasSuffix(tok, "]") {
		if inner := tok[1 : len(tok)-1]; strings.TrimSpace(inner) != "" {
			return expr.Compile(inner, scope)
		}
		return expr.Compile("[]", scope)
	}
	return expr.Compile(tok, scope)
}

func parseFuncHeader(line string) (string, []string, bool) {
	header := strings.TrimSuffix(strings.TrimSpace(strings.TrimPrefix(line, "func ")), "{")
	header = strings.TrimSpace(header)

	open := strings.Index(header, "(")
	if open < 0 || !strings.HasSuffix(header, ")") {
		return "", nil, false
	}

	name := strings.TrimSpace(header[:open])
	if name == "" {
		return "", nil, false
	}

	var params []string
	if inner := strings.TrimSpace(header[open+1 : len(header)-1]); inner != "" {
		for _, p := range strings.Split(inner, ",") {
			params = append(params, strings.TrimSpace(p))
		}
	}
	return name, params, true
}

func identifier(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == '_' || c == '.':
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
		case i > 0 && c >= '0' && c <= '9':
		default:
			return false
		}
	}
	return true
}

// splitArgs breaks an argument list on the commas that sit at nesting depth
// zero, so a call passed as an argument survives intact.
func splitArgs(s string) []string {
	var args []string
	depth, start := 0, 0
	inString := false

	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == '"':
			inString = !inString
		case inString:
		case c == '(' || c == '[':
			depth++
		case c == ')' || c == ']':
			depth--
		case c == ',' && depth == 0:
			args = append(args, strings.TrimSpace(s[start:i]))
			start = i + 1
		}
	}

	if last := strings.TrimSpace(s[start:]); last != "" {
		args = append(args, last)
	}
	return args
}

func parseCallLine(line string) (string, []string, bool) {
	if !strings.HasSuffix(line, ")") {
		return "", nil, false
	}

	open := strings.Index(line, "(")
	if open <= 0 {
		return "", nil, false
	}

	name := strings.TrimSpace(line[:open])
	if !identifier(name) {
		return "", nil, false
	}
	return name, splitArgs(line[open+1 : len(line)-1]), true
}

func (c *compiler) assignment(at int, line, keyword string, global bool) Node {
	rest := strings.TrimPrefix(line, keyword+" ")

	eq := strings.Index(rest, "=")
	if eq < 0 {
		c.pos++
		c.fail(at, line, "%s needs the form: %s x = value", keyword, keyword)
		return nil
	}

	val, err := compileValue(strings.TrimSpace(rest[eq+1:]))
	if err != nil {
		c.pos++
		c.fail(at, line, "%v", err)
		return nil
	}

	name := strings.TrimSpace(rest[:eq])
	if !identifier(name) {
		c.pos++
		c.fail(at, line, "%q is not a valid name", name)
		return nil
	}

	c.pos++
	return &assignNode{name: name, slot: scope.Slot(name), val: val, global: global}
}

func (c *compiler) statement(line string) Node {
	at := c.pos

	switch firstWord(line) {
	case "break":
		c.pos++
		return breakNode{}

	case "continue":
		c.pos++
		return continueNode{}

	case "return":
		c.pos++
		rest := strings.TrimSpace(strings.TrimPrefix(line, "return"))
		if rest == "" {
			return &returnNode{}
		}
		val, err := compileValue(rest)
		if err != nil {
			c.fail(at, line, "%v", err)
			return nil
		}
		return &returnNode{val: val}

	case "import":
		c.pos++
		name := strings.TrimSpace(strings.TrimPrefix(line, "import"))
		if !identifier(name) {
			c.fail(at, line, "import needs a package name")
			return nil
		}
		return &importNode{name: name}

	case "var":
		return c.assignment(at, line, "var", false)

	case "global":
		return c.assignment(at, line, "global", true)

	case "func":
		name, params, ok := parseFuncHeader(line)
		if !ok {
			c.pos++
			c.fail(at, line, "func needs the form: func name(a, b) {")
			return nil
		}
		c.pos++
		userFuncs[name] = &funcDef{params: params, body: c.block(true)}
		return nil

	case "while":
		cond, err := conditionOf(line)
		if err != nil {
			c.pos++
			c.fail(at, line, "%v", err)
			return nil
		}
		c.pos++
		return &whileNode{cond: cond, body: c.block(true)}

	case "repeat":
		header := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(strings.TrimPrefix(line, "repeat")), "{"))
		count, err := compileValue(header)
		if err != nil {
			c.pos++
			c.fail(at, line, "%v", err)
			return nil
		}
		c.pos++
		return &repeatNode{count: count, body: c.block(true)}

	case "each":
		header := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(strings.TrimPrefix(line, "each")), "{"))
		sep := strings.Index(header, " in ")
		if sep < 0 {
			c.pos++
			c.fail(at, line, "each needs the form: each x in xs {")
			return nil
		}

		list, err := compileValue(strings.TrimSpace(header[sep+4:]))
		if err != nil {
			c.pos++
			c.fail(at, line, "%v", err)
			return nil
		}
		c.pos++
		return &eachNode{name: strings.TrimSpace(header[:sep]), list: list, body: c.block(true)}

	case "if":
		cond, err := conditionOf(line)
		if err != nil {
			c.pos++
			c.fail(at, line, "%v", err)
			return nil
		}
		c.pos++
		then := c.block(true)

		var els []Node
		if i := c.nextMeaningful(); i < len(c.lines) {
			next := strings.TrimSpace(c.lines[i])
			switch {
			case strings.HasPrefix(next, "else if"):
				// A chained condition is just an if whose header follows else.
				c.pos = i
				if n := c.statement(strings.TrimSpace(strings.TrimPrefix(next, "else"))); n != nil {
					els = []Node{n}
				}
			case strings.HasPrefix(next, "else"):
				c.pos = i + 1
				els = c.block(true)
			}
		}
		return &ifNode{cond: cond, then: then, els: els}
	}

	if name, args, ok := parseCallLine(line); ok {
		c.pos++

		exprs := make([]expr.Expr, 0, len(args))
		for _, a := range args {
			e, err := compileValue(a)
			if err != nil {
				c.fail(at, line, "%v", err)
				return nil
			}
			exprs = append(exprs, e)
		}
		return &callNode{name: name, args: exprs}
	}

	c.pos++
	c.fail(at, line, "not a statement")
	return nil
}

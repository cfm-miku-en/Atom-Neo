package expr

import (
	"fmt"
	"strconv"
)

type tokKind uint8

const (
	tokEOF tokKind = iota
	tokNum
	tokStr
	tokIdent
	tokOp
	tokLParen
	tokRParen
	tokComma
	tokLBracket
	tokRBracket
	tokColon
)

type token struct {
	kind tokKind
	text string
	num  float64
}

func identStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func identPart(c byte) bool {
	// A dot continues an identifier so module calls such as web.method() read
	// as one name; numbers are handled before this is ever reached.
	return identStart(c) || c == '.' || (c >= '0' && c <= '9')
}

func lex(src string) ([]token, error) {
	toks := make([]token, 0, 8)

	for i := 0; i < len(src); {
		c := src[i]

		switch {
		case c == ' ' || c == '\t':
			i++

		case c >= '0' && c <= '9':
			j := i
			for j < len(src) && ((src[j] >= '0' && src[j] <= '9') || src[j] == '.') {
				j++
			}
			f, err := strconv.ParseFloat(src[i:j], 64)
			if err != nil {
				return nil, fmt.Errorf("bad number %q", src[i:j])
			}
			toks = append(toks, token{kind: tokNum, num: f})
			i = j

		case c == '"':
			j := i + 1
			for j < len(src) && src[j] != '"' {
				if src[j] == '\\' {
					j++
				}
				j++
			}
			if j >= len(src) {
				return nil, fmt.Errorf("unterminated string")
			}
			toks = append(toks, token{kind: tokStr, text: Unescape(src[i+1 : j])})
			i = j + 1

		case identStart(c):
			j := i
			for j < len(src) && identPart(src[j]) {
				j++
			}
			toks = append(toks, token{kind: tokIdent, text: src[i:j]})
			i = j

		case c == '(':
			toks = append(toks, token{kind: tokLParen})
			i++

		case c == ')':
			toks = append(toks, token{kind: tokRParen})
			i++

		case c == ',':
			toks = append(toks, token{kind: tokComma})
			i++

		case c == '[':
			toks = append(toks, token{kind: tokLBracket})
			i++

		case c == ']':
			toks = append(toks, token{kind: tokRBracket})
			i++

		case c == ':':
			toks = append(toks, token{kind: tokColon})
			i++

		default:
			if i+1 < len(src) {
				switch two := src[i : i+2]; two {
				case "==", "!=", "<=", ">=", "&&", "||", "?=":
					// ?= is Atom's spelling of !=
					if two == "?=" {
						two = "!="
					}
					toks = append(toks, token{kind: tokOp, text: two})
					i += 2
					continue
				}
			}
			switch c {
			case '+', '-', '*', '/', '%', '<', '>', '!', '@':
				toks = append(toks, token{kind: tokOp, text: string(c)})
				i++
			default:
				return nil, fmt.Errorf("unexpected character %q", string(c))
			}
		}
	}

	return append(toks, token{kind: tokEOF}), nil
}

var precedence = map[string]int{
	"||": 1,
	"&&": 2,
	"==": 3, "!=": 3,
	"<": 4, ">": 4, "<=": 4, ">=": 4,
	"+": 5, "-": 5,
	"*": 6, "/": 6, "%": 6,
	"@": 7,
}

var binaryOps = map[string]op{
	"+": opAdd, "-": opSub, "*": opMul, "/": opDiv, "%": opMod,
	"==": opEq, "!=": opNe, "<": opLt, ">": opGt, "<=": opLe, ">=": opGe,
	"&&": opAnd, "||": opOr,
	"@": opIndex,
}

type parser struct {
	toks  []token
	pos   int
	scope *Scope
}

func (p *parser) peek() token { return p.toks[p.pos] }

func (p *parser) next() token {
	t := p.toks[p.pos]
	p.pos++
	return t
}

func (p *parser) parseBinary(minPrec int) (Expr, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for {
		t := p.peek()
		if t.kind != tokOp {
			break
		}
		prec, ok := precedence[t.text]
		if !ok || prec < minPrec {
			break
		}
		p.next()

		right, err := p.parseBinary(prec + 1)
		if err != nil {
			return nil, err
		}
		left = binary(binaryOps[t.text], left, right)
	}
	return left, nil
}

func (p *parser) parseUnary() (Expr, error) {
	t := p.peek()
	if t.kind == tokOp && (t.text == "!" || t.text == "-") {
		p.next()
		x, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		o := opNot
		if t.text == "-" {
			o = opNeg
		}
		return unary(o, x), nil
	}
	return p.parsePrimary()
}

// mapAhead reports whether the next two tokens look like a map key followed by
// its colon, which is what separates [a: 1] from [a, 1].
func (p *parser) mapAhead() bool {
	if p.pos+1 >= len(p.toks) {
		return false
	}
	k := p.toks[p.pos].kind
	return (k == tokIdent || k == tokStr) && p.toks[p.pos+1].kind == tokColon
}

func (p *parser) parseMapEntries(closing bool) (Expr, error) {
	var pairs []mapPair

	for {
		key := p.next()
		if key.kind != tokIdent && key.kind != tokStr {
			return nil, fmt.Errorf("map key must be a name or a string")
		}
		if p.next().kind != tokColon {
			return nil, fmt.Errorf("expected : after map key %q", key.text)
		}

		val, err := p.parseBinary(1)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, mapPair{key: key.text, val: val})

		if p.peek().kind != tokComma {
			break
		}
		p.next()
	}

	if closing && p.next().kind != tokRBracket {
		return nil, fmt.Errorf("missing closing bracket in map")
	}
	return dict(pairs), nil
}

func (p *parser) parseCall(name string) (Expr, error) {
	p.next()

	var args []Expr
	if p.peek().kind != tokRParen {
		for {
			a, err := p.parseBinary(1)
			if err != nil {
				return nil, err
			}
			args = append(args, a)

			if p.peek().kind != tokComma {
				break
			}
			p.next()
		}
	}

	if p.next().kind != tokRParen {
		return nil, fmt.Errorf("missing closing parenthesis in call to %s", name)
	}
	return call(name, args), nil
}

func (p *parser) parsePrimary() (Expr, error) {
	t := p.next()

	switch t.kind {
	case tokNum:
		return literal(Num(t.num)), nil
	case tokStr:
		return literal(Str(t.text)), nil
	case tokIdent:
		switch t.text {
		case "true":
			return literal(Boolean(true)), nil
		case "false":
			return literal(Boolean(false)), nil
		}
		if p.peek().kind == tokLParen {
			return p.parseCall(t.text)
		}
		return variable(p.scope.Resolve(t.text)), nil
	case tokLParen:
		e, err := p.parseBinary(1)
		if err != nil {
			return nil, err
		}
		if p.next().kind != tokRParen {
			return nil, fmt.Errorf("missing closing parenthesis")
		}
		return e, nil

	case tokLBracket:
		if p.peek().kind == tokColon && p.pos+1 < len(p.toks) && p.toks[p.pos+1].kind == tokRBracket {
			p.next()
			p.next()
			return dict(nil), nil
		}
		if p.mapAhead() {
			return p.parseMapEntries(true)
		}

		var items []Expr
		if p.peek().kind != tokRBracket {
			for {
				item, err := p.parseBinary(1)
				if err != nil {
					return nil, err
				}
				items = append(items, item)

				if p.peek().kind != tokComma {
					break
				}
				p.next()
			}
		}
		if p.next().kind != tokRBracket {
			return nil, fmt.Errorf("missing closing bracket")
		}
		return list(items), nil
	}
	return nil, fmt.Errorf("unexpected token in expression")
}

func Compile(src string, scope *Scope) (Expr, error) {
	toks, err := lex(src)
	if err != nil {
		return nil, err
	}

	p := &parser{toks: toks, scope: scope}

	// [:] is an empty map; the brackets are gone by now so only the colon is left.
	if p.peek().kind == tokColon && p.toks[p.pos+1].kind == tokEOF {
		return dict(nil), nil
	}

	// The outer brackets are stripped before this point, so a map literal
	// arrives as bare key: value pairs.
	if p.mapAhead() {
		e, err := p.parseMapEntries(false)
		if err != nil {
			return nil, err
		}
		if p.peek().kind != tokEOF {
			return nil, fmt.Errorf("unexpected trailing input in %q", src)
		}
		return e, nil
	}

	e, err := p.parseBinary(1)
	if err != nil {
		return nil, err
	}

	// Commas at the top of a bracket make it a list rather than a single value.
	if p.peek().kind == tokComma {
		items := []Expr{e}
		for p.peek().kind == tokComma {
			p.next()
			next, err := p.parseBinary(1)
			if err != nil {
				return nil, err
			}
			items = append(items, next)
		}
		e = list(items)
	}

	if p.peek().kind != tokEOF {
		return nil, fmt.Errorf("unexpected trailing input in %q", src)
	}
	return e, nil
}

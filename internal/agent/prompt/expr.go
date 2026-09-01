package prompt

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type tokenKind int

const (
	tkEOF tokenKind = iota
	tkIdent
	tkNumber
	tkString
	tkTrue
	tkFalse
	tkNil
	tkOp
	tkLParen
	tkRParen
)

type token struct {
	kind tokenKind
	text string
	pos  int
}

var operators = []string{"==", "!=", "<=", ">=", "&&", "||", "<", ">", "+", "-", "*", "/", "!", ".", ","}

func tokenize(input string) ([]token, error) {
	var toks []token
	for i := 0; i < len(input); {
		ch := input[i]
		switch {
		case ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r':
			i++
		case ch == '(':
			toks = append(toks, token{kind: tkLParen, text: "(", pos: i})
			i++
		case ch == ')':
			toks = append(toks, token{kind: tkRParen, text: ")", pos: i})
			i++
		case ch == '"' || ch == '\'':
			s, next, err := scanString(input, i)
			if err != nil {
				return nil, err
			}
			toks = append(toks, token{kind: tkString, text: s, pos: i})
			i = next
		case ch >= '0' && ch <= '9':
			s, next, err := scanNumber(input, i)
			if err != nil {
				return nil, err
			}
			toks = append(toks, token{kind: tkNumber, text: s, pos: i})
			i = next
		case isIdentStart(ch):
			s, next := scanIdent(input, i)
			switch s {
			case "true":
				toks = append(toks, token{kind: tkTrue, text: s, pos: i})
			case "false":
				toks = append(toks, token{kind: tkFalse, text: s, pos: i})
			case "nil", "null":
				toks = append(toks, token{kind: tkNil, text: s, pos: i})
			default:
				toks = append(toks, token{kind: tkIdent, text: s, pos: i})
			}
			i = next
		default:
			matched := false
			for _, op := range operators {
				if strings.HasPrefix(input[i:], op) {
					toks = append(toks, token{kind: tkOp, text: op, pos: i})
					i += len(op)
					matched = true
					break
				}
			}
			if !matched {
				return nil, fmt.Errorf("prompt: expression: unexpected character %q at position %d", string(ch), i)
			}
		}
	}
	toks = append(toks, token{kind: tkEOF, text: "", pos: len(input)})
	return toks, nil
}

func isIdentStart(ch byte) bool {
	return ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func scanIdent(input string, start int) (string, int) {
	i := start
	for i < len(input) {
		ch := input[i]
		if isIdentStart(ch) || (ch >= '0' && ch <= '9') {
			i++
			continue
		}
		break
	}
	return input[start:i], i
}

func scanNumber(input string, start int) (string, int, error) {
	i := start
	for i < len(input) && input[i] >= '0' && input[i] <= '9' {
		i++
	}
	if i < len(input) && input[i] == '.' {
		if i+1 >= len(input) || input[i+1] < '0' || input[i+1] > '9' {
			return "", 0, fmt.Errorf("prompt: expression: malformed number at position %d", start)
		}
		i++
		for i < len(input) && input[i] >= '0' && input[i] <= '9' {
			i++
		}
	}
	if i < len(input) && isIdentStart(input[i]) {
		return "", 0, fmt.Errorf("prompt: expression: malformed number at position %d", start)
	}
	if _, err := strconv.ParseFloat(input[start:i], 64); err != nil {
		return "", 0, fmt.Errorf("prompt: expression: malformed number at position %d", start)
	}
	return input[start:i], i, nil
}

func scanString(input string, start int) (string, int, error) {
	quote := input[start]
	var sb strings.Builder
	i := start + 1
	for i < len(input) {
		ch := input[i]
		if ch == quote {
			return sb.String(), i + 1, nil
		}
		if ch == '\\' {
			if i+1 >= len(input) {
				return "", 0, fmt.Errorf("prompt: expression: unterminated escape at position %d", i)
			}
			switch input[i+1] {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case 'r':
				sb.WriteByte('\r')
			case '\\':
				sb.WriteByte('\\')
			case '"', '\'':
				sb.WriteByte(input[i+1])
			default:
				return "", 0, fmt.Errorf("prompt: expression: invalid escape \\%c at position %d", input[i+1], i)
			}
			i += 2
			continue
		}
		sb.WriteByte(ch)
		i++
	}
	return "", 0, fmt.Errorf("prompt: expression: unterminated string starting at position %d", start)
}

type nodeKind int

const (
	nkLiteral nodeKind = iota
	nkPath
	nkCall
	nkUnary
	nkBinary
)

type node struct {
	kind  nodeKind
	lit   any
	path  []string
	name  string
	args  []*node
	op    string
	left  *node
	right *node
}

type parser struct {
	toks []token
	pos  int
}

func parse(input string) (*node, error) {
	toks, err := tokenize(input)
	if err != nil {
		return nil, err
	}
	p := &parser{toks: toks}
	n, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if t := p.peek(); t.kind != tkEOF {
		return nil, fmt.Errorf("prompt: expression: unexpected token %q at position %d", t.text, t.pos)
	}
	return n, nil
}

func (p *parser) peek() token {
	return p.toks[p.pos]
}

func (p *parser) next() token {
	t := p.toks[p.pos]
	if t.kind != tkEOF {
		p.pos++
	}
	return t
}

func (p *parser) matchOp(op string) bool {
	if p.peek().kind == tkOp && p.peek().text == op {
		p.pos++
		return true
	}
	return false
}

func (p *parser) parseOr() (*node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.matchOp("||") {
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &node{kind: nkBinary, op: "||", left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseAnd() (*node, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}
	for p.matchOp("&&") {
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		left = &node{kind: nkBinary, op: "&&", left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseNot() (*node, error) {
	if p.matchOp("!") {
		inner, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		return &node{kind: nkUnary, op: "!", left: inner}, nil
	}
	return p.parseCmp()
}

var cmpOps = []string{"==", "!=", "<=", ">=", "<", ">"}

func (p *parser) parseCmp() (*node, error) {
	left, err := p.parseSum()
	if err != nil {
		return nil, err
	}
	for _, op := range cmpOps {
		if p.matchOp(op) {
			right, err := p.parseSum()
			if err != nil {
				return nil, err
			}
			return &node{kind: nkBinary, op: op, left: left, right: right}, nil
		}
	}
	return left, nil
}

func (p *parser) parseSum() (*node, error) {
	left, err := p.parseTerm()
	if err != nil {
		return nil, err
	}
	for p.matchOp("+") || p.matchOp("-") {
		op := p.toks[p.pos-1].text
		right, err := p.parseTerm()
		if err != nil {
			return nil, err
		}
		left = &node{kind: nkBinary, op: op, left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseTerm() (*node, error) {
	left, err := p.parseFactor()
	if err != nil {
		return nil, err
	}
	for p.matchOp("*") || p.matchOp("/") {
		op := p.toks[p.pos-1].text
		right, err := p.parseFactor()
		if err != nil {
			return nil, err
		}
		left = &node{kind: nkBinary, op: op, left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseFactor() (*node, error) {
	if p.matchOp("-") {
		inner, err := p.parseFactor()
		if err != nil {
			return nil, err
		}
		return &node{kind: nkUnary, op: "-", left: inner}, nil
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() (*node, error) {
	t := p.peek()
	switch t.kind {
	case tkNumber, tkString, tkTrue, tkFalse, tkNil:
		p.next()
		var lit any
		switch t.kind {
		case tkNumber:
			lit, _ = strconv.ParseFloat(t.text, 64)
		case tkString:
			lit = t.text
		case tkTrue:
			lit = true
		case tkFalse:
			lit = false
		}
		return &node{kind: nkLiteral, lit: lit}, nil
	case tkIdent:
		name := t.text
		p.next()
		if p.peek().kind == tkLParen {
			p.next()
			var args []*node
			if p.peek().kind == tkRParen {
				p.next()
				return &node{kind: nkCall, name: name}, nil
			}
			for {
				arg, err := p.parseOr()
				if err != nil {
					return nil, err
				}
				args = append(args, arg)
				if p.peek().kind == tkRParen {
					p.next()
					break
				}
				if !p.matchOp(",") {
					return nil, fmt.Errorf("prompt: expression: expected ',' or ')' at position %d", p.peek().pos)
				}
			}
			return &node{kind: nkCall, name: name, args: args}, nil
		}
		path := []string{name}
		for p.matchOp(".") {
			id := p.next()
			if id.kind != tkIdent {
				return nil, fmt.Errorf("prompt: expression: expected identifier after '.' at position %d", id.pos)
			}
			path = append(path, id.text)
		}
		return &node{kind: nkPath, path: path}, nil
	case tkLParen:
		p.next()
		n, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.next().kind != tkRParen {
			return nil, fmt.Errorf("prompt: expression: expected ')' at position %d", p.peek().pos)
		}
		return n, nil
	}
	return nil, fmt.Errorf("prompt: expression: unexpected token %q at position %d", t.text, t.pos)
}

func evalBool(expr string, vars Vars) (bool, error) {
	n, err := parse(expr)
	if err != nil {
		return false, err
	}
	v, err := evalNode(n, vars)
	if err != nil {
		return false, err
	}
	b, ok := v.(bool)
	if !ok {
		return false, fmt.Errorf("prompt: expression %q must evaluate to a boolean, got %T", expr, v)
	}
	return b, nil
}

func evalValue(expr string, vars Vars) (any, error) {
	n, err := parse(expr)
	if err != nil {
		return nil, err
	}
	return evalNode(n, vars)
}

func isComparison(n *node) bool {
	return n.kind == nkBinary && n.op != "&&" && n.op != "||" && n.op != "+" &&
		n.op != "-" && n.op != "*" && n.op != "/"
}

func evalNode(n *node, vars Vars) (any, error) {
	switch n.kind {
	case nkLiteral:
		return n.lit, nil
	case nkPath:
		key := strings.Join(n.path, ".")
		v, ok := vars.Get(key)
		if !ok {
			return nil, fmt.Errorf("%w: %q", ErrMissingVar, key)
		}
		return v, nil
	case nkCall:
		return evalCall(n, vars)
	case nkUnary:
		v, err := evalNode(n.left, vars)
		if err != nil {
			return nil, err
		}
		switch n.op {
		case "!":
			b, ok := v.(bool)
			if !ok {
				return nil, fmt.Errorf("prompt: expression: cannot apply \"!\" to %T", v)
			}
			return !b, nil
		case "-":
			f, ok := asNumber(v)
			if !ok {
				return nil, fmt.Errorf("prompt: expression: cannot negate %T", v)
			}
			return -f, nil
		}
	case nkBinary:
		switch n.op {
		case "&&":
			l, err := evalNode(n.left, vars)
			if err != nil {
				return nil, err
			}
			lb, ok := l.(bool)
			if !ok {
				return nil, fmt.Errorf("prompt: expression: cannot apply \"&&\" to %T", l)
			}
			if !lb {
				return false, nil
			}
			r, err := evalNode(n.right, vars)
			if err != nil {
				return nil, err
			}
			rb, ok := r.(bool)
			if !ok {
				return nil, fmt.Errorf("prompt: expression: cannot apply \"&&\" to %T", r)
			}
			return rb, nil
		case "||":
			l, err := evalNode(n.left, vars)
			if err != nil {
				return nil, err
			}
			lb, ok := l.(bool)
			if !ok {
				return nil, fmt.Errorf("prompt: expression: cannot apply \"||\" to %T", l)
			}
			if lb {
				return true, nil
			}
			r, err := evalNode(n.right, vars)
			if err != nil {
				return nil, err
			}
			rb, ok := r.(bool)
			if !ok {
				return nil, fmt.Errorf("prompt: expression: cannot apply \"||\" to %T", r)
			}
			return rb, nil
		}
		l, err := evalNode(n.left, vars)
		if err != nil {
			return nil, err
		}
		r, err := evalNode(n.right, vars)
		if err != nil {
			return nil, err
		}
		return applyBinary(n.op, l, r)
	}
	return nil, errors.New("prompt: expression: unknown node kind")
}

func evalCall(n *node, vars Vars) (any, error) {
	switch n.name {
	case "defined":
		if len(n.args) != 1 {
			return nil, fmt.Errorf("prompt: expression: defined() expects exactly one argument, got %d", len(n.args))
		}
		arg := n.args[0]
		var key string
		if arg.kind == nkPath {
			key = strings.Join(arg.path, ".")
		} else {
			v, err := evalNode(arg, vars)
			if err != nil {
				return nil, err
			}
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("prompt: expression: defined() expects a string path, got %T", v)
			}
			key = s
		}
		return vars.Has(key), nil
	case "len":
		if len(n.args) != 1 {
			return nil, fmt.Errorf("prompt: expression: len() expects exactly one argument, got %d", len(n.args))
		}
		v, err := resolveCallArg(n.args[0], vars)
		if err != nil {
			return nil, err
		}
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
			return float64(rv.Len()), nil
		}
		return nil, fmt.Errorf("prompt: expression: len() cannot measure %T", v)
	case "empty":
		if len(n.args) != 1 {
			return nil, fmt.Errorf("prompt: expression: empty() expects exactly one argument, got %d", len(n.args))
		}
		v, err := resolveCallArg(n.args[0], vars)
		if err != nil {
			return nil, err
		}
		if v == nil {
			return true, nil
		}
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
			return rv.Len() == 0, nil
		}
		return nil, fmt.Errorf("prompt: expression: empty() cannot measure %T", v)
	}
	return nil, fmt.Errorf("prompt: expression: unknown function %q", n.name)
}

func resolveCallArg(arg *node, vars Vars) (any, error) {
	if arg.kind == nkPath {
		key := strings.Join(arg.path, ".")
		v, ok := vars.Get(key)
		if !ok {
			return nil, fmt.Errorf("%w: %q", ErrMissingVar, key)
		}
		return v, nil
	}
	v, err := evalNode(arg, vars)
	if err != nil {
		return nil, err
	}
	if s, isStr := v.(string); isStr {
		got, ok := vars.Get(s)
		if !ok {
			return nil, fmt.Errorf("%w: %q", ErrMissingVar, s)
		}
		return got, nil
	}
	return v, nil
}

func applyBinary(op string, l, r any) (any, error) {
	switch op {
	case "==":
		return equalValues(l, r), nil
	case "!=":
		return !equalValues(l, r), nil
	case "<", "<=", ">", ">=":
		if lf, lok := asNumber(l); lok {
			if rf, rok := asNumber(r); rok {
				return compareNumbers(op, lf, rf), nil
			}
		}
		if ls, lok := l.(string); lok {
			if rs, rok := r.(string); rok {
				switch op {
				case "<":
					return ls < rs, nil
				case "<=":
					return ls <= rs, nil
				case ">":
					return ls > rs, nil
				default:
					return ls >= rs, nil
				}
			}
		}
		return nil, fmt.Errorf("prompt: expression: cannot compare %T and %T with %q", l, r, op)
	case "+":
		if lf, lok := asNumber(l); lok {
			if rf, rok := asNumber(r); rok {
				return lf + rf, nil
			}
		}
		if ls, lok := l.(string); lok {
			if rs, rok := r.(string); rok {
				return ls + rs, nil
			}
		}
		return nil, fmt.Errorf("prompt: expression: cannot add %T and %T", l, r)
	case "-", "*", "/":
		lf, lok := asNumber(l)
		rf, rok := asNumber(r)
		if !lok || !rok {
			return nil, fmt.Errorf("prompt: expression: cannot apply %q to %T and %T", op, l, r)
		}
		switch op {
		case "-":
			return lf - rf, nil
		case "*":
			return lf * rf, nil
		default:
			if rf == 0 {
				return nil, errors.New("prompt: expression: division by zero")
			}
			return lf / rf, nil
		}
	}
	return nil, fmt.Errorf("prompt: expression: unknown operator %q", op)
}

func compareNumbers(op string, l, r float64) bool {
	switch op {
	case "<":
		return l < r
	case "<=":
		return l <= r
	case ">":
		return l > r
	default:
		return l >= r
	}
}

func equalValues(l, r any) bool {
	if l == nil || r == nil {
		return l == nil && r == nil
	}
	if lf, lok := asNumber(l); lok {
		if rf, rok := asNumber(r); rok {
			return lf == rf
		}
	}
	return reflect.DeepEqual(l, r)
}

func asNumber(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case int32:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint64:
		return float64(n), true
	}
	return 0, false
}

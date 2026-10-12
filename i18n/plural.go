package i18n

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// PluralForms is a language's plural rule, from a PO file's Plural-Forms
// header, e.g. "nplurals=2; plural=(n != 1);".
type PluralForms struct {
	N    int
	expr node
}

// Index returns which plural form goes with n.
func (p *PluralForms) Index(n int) int {
	i := p.expr.eval(n)

	// A rule that gives a form the language doesn't have is a mistake in the
	// file; the last form is the likeliest to fit.
	if i < 0 || i >= p.N {
		return p.N - 1
	}

	return i
}

// ParsePluralForms reads a Plural-Forms header's value.
func ParsePluralForms(header string) (*PluralForms, error) {
	var count, expr string

	for part := range strings.SplitSeq(header, ";") {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}

		switch strings.TrimSpace(key) {
		case "nplurals":
			count = strings.TrimSpace(value)
		case "plural":
			expr = strings.TrimSpace(value)
		}
	}

	n, err := strconv.Atoi(count)
	if err != nil || n < 1 {
		return nil, fmt.Errorf("plural forms %q: nplurals isn't a number", header)
	}

	node, err := parseExpr(expr)
	if err != nil {
		return nil, fmt.Errorf("plural forms %q: %w", header, err)
	}

	return &PluralForms{N: n, expr: node}, nil
}

// The plural expression is C, with n as its only variable. Its value is an
// unsigned long in gettext, but for counts of things an int does.
type node interface {
	eval(n int) int
}

type numberNode int

func (v numberNode) eval(int) int { return int(v) }

type varNode struct{}

func (varNode) eval(n int) int { return n }

type unaryNode struct {
	op string
	x  node
}

func (u unaryNode) eval(n int) int {
	if u.op == "!" {
		return boolInt(u.x.eval(n) == 0)
	}

	return -u.x.eval(n)
}

type binaryNode struct {
	op   string
	x, y node
}

func (b binaryNode) eval(n int) int {
	x := b.x.eval(n)

	// && and || don't evaluate their right side when the left decides.
	switch b.op {
	case "&&":
		return boolInt(x != 0 && b.y.eval(n) != 0)
	case "||":
		return boolInt(x != 0 || b.y.eval(n) != 0)
	}

	y := b.y.eval(n)

	switch b.op {
	case "*":
		return x * y
	case "/", "%":
		// Division by zero is a mistake in the file, not a reason to crash.
		if y == 0 {
			return 0
		}

		if b.op == "/" {
			return x / y
		}

		return x % y
	case "+":
		return x + y
	case "-":
		return x - y
	case "<":
		return boolInt(x < y)
	case "<=":
		return boolInt(x <= y)
	case ">":
		return boolInt(x > y)
	case ">=":
		return boolInt(x >= y)
	case "==":
		return boolInt(x == y)
	default: // "!="
		return boolInt(x != y)
	}
}

type conditionNode struct {
	cond, then, otherwise node
}

func (c conditionNode) eval(n int) int {
	if c.cond.eval(n) != 0 {
		return c.then.eval(n)
	}

	return c.otherwise.eval(n)
}

func boolInt(b bool) int {
	if b {
		return 1
	}

	return 0
}

// Binary operators by precedence, loosest first, as in C.
var precedence = map[string]int{
	"||": 1,
	"&&": 2,
	"==": 3, "!=": 3,
	"<": 4, "<=": 4, ">": 4, ">=": 4,
	"+": 5, "-": 5,
	"*": 6, "/": 6, "%": 6,
}

var twoCharOps = map[string]bool{"&&": true, "||": true, "==": true, "!=": true, "<=": true, ">=": true}

type exprParser struct {
	tokens []string
	pos    int
}

func parseExpr(s string) (node, error) {
	tokens, err := tokenize(s)
	if err != nil {
		return nil, err
	}

	p := &exprParser{tokens: tokens}

	n, err := p.condition()
	if err != nil {
		return nil, err
	}

	if p.pos < len(p.tokens) {
		return nil, fmt.Errorf("unexpected %q", p.tokens[p.pos])
	}

	return n, nil
}

func tokenize(s string) ([]string, error) {
	var tokens []string

	for i := 0; i < len(s); {
		c := s[i]

		switch {
		case c == ' ' || c == '\t' || c == '\n':
			i++
		case c >= '0' && c <= '9':
			j := i
			for j < len(s) && s[j] >= '0' && s[j] <= '9' {
				j++
			}

			tokens = append(tokens, s[i:j])
			i = j
		case c == 'n':
			tokens = append(tokens, "n")
			i++
		case i+1 < len(s) && twoCharOps[s[i:i+2]]:
			tokens = append(tokens, s[i:i+2])
			i += 2
		case strings.IndexByte("()?:!<>+-*/%", c) >= 0:
			tokens = append(tokens, string(c))
			i++
		default:
			return nil, fmt.Errorf("unexpected %q", c)
		}
	}

	if len(tokens) == 0 {
		return nil, errors.New("no plural expression")
	}

	return tokens, nil
}

func (p *exprParser) peek() string {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}

	return ""
}

func (p *exprParser) condition() (node, error) {
	cond, err := p.binary(1)
	if err != nil {
		return nil, err
	}

	if p.peek() != "?" {
		return cond, nil
	}

	p.pos++

	then, err := p.condition()
	if err != nil {
		return nil, err
	}

	if p.peek() != ":" {
		return nil, errors.New(`"?" without ":"`)
	}

	p.pos++

	otherwise, err := p.condition()
	if err != nil {
		return nil, err
	}

	return conditionNode{cond, then, otherwise}, nil
}

func (p *exprParser) binary(minPrecedence int) (node, error) {
	x, err := p.unary()
	if err != nil {
		return nil, err
	}

	for {
		op := p.peek()

		prec, ok := precedence[op]
		if !ok || prec < minPrecedence {
			return x, nil
		}

		p.pos++

		y, err := p.binary(prec + 1)
		if err != nil {
			return nil, err
		}

		x = binaryNode{op, x, y}
	}
}

func (p *exprParser) unary() (node, error) {
	token := p.peek()
	p.pos++

	switch {
	case token == "!" || token == "-":
		x, err := p.unary()
		if err != nil {
			return nil, err
		}

		return unaryNode{token, x}, nil
	case token == "(":
		x, err := p.condition()
		if err != nil {
			return nil, err
		}

		if p.peek() != ")" {
			return nil, errors.New(`"(" without ")"`)
		}

		p.pos++

		return x, nil
	case token == "n":
		return varNode{}, nil
	case token != "" && token[0] >= '0' && token[0] <= '9':
		v, err := strconv.Atoi(token)
		if err != nil {
			return nil, err
		}

		return numberNode(v), nil
	case token == "":
		return nil, errors.New("the expression ends too soon")
	default:
		return nil, fmt.Errorf("unexpected %q", token)
	}
}

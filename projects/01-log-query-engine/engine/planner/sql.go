// Package planner parses SQL into a LogicalPlan AST.
package planner

import (
	"fmt"
	"strings"
	"unicode"
)

// ─────────────────────────────────────────────────────────────
// AST Node Types
// ─────────────────────────────────────────────────────────────

// SelectStmt is the top-level AST node for a SELECT query.
type SelectStmt struct {
	Columns   []SelectExpr // SELECT list
	From      string       // table name (file alias)
	Where     Expr         // WHERE clause (nil if absent)
	GroupBy   []Expr       // GROUP BY expressions
	OrderBy   []OrderExpr  // ORDER BY expressions
	Limit     int64        // LIMIT (0 = no limit)
	HasLimit  bool
}

// SelectExpr is a single item in the SELECT list.
type SelectExpr struct {
	Expr  Expr
	Alias string // AS alias
}

// OrderExpr is a sort key with direction.
type OrderExpr struct {
	Expr Expr
	Desc bool
}

// Expr is the expression interface.
type Expr interface {
	exprNode()
	String() string
}

// ColumnRef is a reference to a column by name.
type ColumnRef struct{ Name string }

func (c *ColumnRef) exprNode() {}
func (c *ColumnRef) String() string { return c.Name }

// Literal is a constant value.
type Literal struct{ Value any }

func (l *Literal) exprNode() {}
func (l *Literal) String() string { return fmt.Sprintf("%v", l.Value) }

// BinaryExpr is a binary operation (=, !=, >, <, AND, OR, LIKE, IN).
type BinaryExpr struct {
	Left  Expr
	Op    string
	Right Expr
}

func (b *BinaryExpr) exprNode() {}
func (b *BinaryExpr) String() string {
	return fmt.Sprintf("(%s %s %s)", b.Left.String(), b.Op, b.Right.String())
}

// FuncCallExpr is an aggregate/scalar function call.
type FuncCallExpr struct {
	Name string
	Args []Expr
	Star bool // COUNT(*)
}

func (f *FuncCallExpr) exprNode() {}
func (f *FuncCallExpr) String() string {
	if f.Star {
		return f.Name + "(*)"
	}
	args := make([]string, len(f.Args))
	for i, a := range f.Args {
		args[i] = a.String()
	}
	return f.Name + "(" + strings.Join(args, ", ") + ")"
}

// UnaryExpr is a unary NOT expression.
type UnaryExpr struct {
	Op   string
	Expr Expr
}

func (u *UnaryExpr) exprNode() {}
func (u *UnaryExpr) String() string { return u.Op + " " + u.Expr.String() }

// ─────────────────────────────────────────────────────────────
// Token Types
// ─────────────────────────────────────────────────────────────

type tokenKind int

const (
	tokEOF tokenKind = iota
	tokIdent
	tokString
	tokNumber
	tokStar
	tokComma
	tokLParen
	tokRParen
	tokOp   // =, !=, <, >, <=, >=
	tokSemi
)

type token struct {
	kind tokenKind
	val  string
}

// ─────────────────────────────────────────────────────────────
// Lexer
// ─────────────────────────────────────────────────────────────

type lexer struct {
	input []rune
	pos   int
}

func newLexer(input string) *lexer {
	return &lexer{input: []rune(input)}
}

func (l *lexer) peek() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *lexer) next() rune {
	ch := l.peek()
	l.pos++
	return ch
}

func (l *lexer) skipWS() {
	for l.pos < len(l.input) && unicode.IsSpace(l.input[l.pos]) {
		l.pos++
	}
}

// NextToken returns the next token from the input.
func (l *lexer) NextToken() token {
	l.skipWS()
	if l.pos >= len(l.input) {
		return token{kind: tokEOF}
	}
	ch := l.peek()

	// String literal
	if ch == '\'' || ch == '"' {
		quote := l.next()
		var sb strings.Builder
		for l.pos < len(l.input) {
			c := l.next()
			if c == quote {
				break
			}
			if c == '\\' && l.pos < len(l.input) {
				c = l.next()
			}
			sb.WriteRune(c)
		}
		return token{kind: tokString, val: sb.String()}
	}

	// Number
	if unicode.IsDigit(ch) || (ch == '-' && l.pos+1 < len(l.input) && unicode.IsDigit(l.input[l.pos+1])) {
		var sb strings.Builder
		sb.WriteRune(l.next())
		for l.pos < len(l.input) && (unicode.IsDigit(l.peek()) || l.peek() == '.') {
			sb.WriteRune(l.next())
		}
		return token{kind: tokNumber, val: sb.String()}
	}

	// Operators
	switch ch {
	case '*':
		l.next()
		return token{kind: tokStar, val: "*"}
	case ',':
		l.next()
		return token{kind: tokComma, val: ","}
	case '(':
		l.next()
		return token{kind: tokLParen, val: "("}
	case ')':
		l.next()
		return token{kind: tokRParen, val: ")"}
	case ';':
		l.next()
		return token{kind: tokSemi, val: ";"}
	case '!':
		l.next()
		if l.peek() == '=' {
			l.next()
			return token{kind: tokOp, val: "!="}
		}
		return token{kind: tokOp, val: "!"}
	case '<':
		l.next()
		if l.peek() == '=' {
			l.next()
			return token{kind: tokOp, val: "<="}
		}
		return token{kind: tokOp, val: "<"}
	case '>':
		l.next()
		if l.peek() == '=' {
			l.next()
			return token{kind: tokOp, val: ">="}
		}
		return token{kind: tokOp, val: ">"}
	case '=':
		l.next()
		return token{kind: tokOp, val: "="}
	}

	// Identifier or keyword
	if unicode.IsLetter(ch) || ch == '_' {
		var sb strings.Builder
		for l.pos < len(l.input) && (unicode.IsLetter(l.peek()) || unicode.IsDigit(l.peek()) || l.peek() == '_' || l.peek() == '-' || l.peek() == '.') {
			sb.WriteRune(l.next())
		}
		return token{kind: tokIdent, val: sb.String()}
	}

	// Skip unknown character
	l.next()
	return l.NextToken()
}

// ─────────────────────────────────────────────────────────────
// Parser
// ─────────────────────────────────────────────────────────────

// Parser is a recursive-descent SQL parser.
type Parser struct {
	lex     *lexer
	current token
	peeked  bool
}

// NewParser creates a parser for the given SQL string.
func NewParser(sql string) *Parser {
	p := &Parser{lex: newLexer(sql)}
	p.advance()
	return p
}

func (p *Parser) advance() token {
	t := p.current
	p.current = p.lex.NextToken()
	return t
}

func (p *Parser) expect(kind tokenKind) (token, error) {
	t := p.current
	if t.kind != kind {
		return t, fmt.Errorf("expected token kind %d, got %q", kind, t.val)
	}
	p.advance()
	return t, nil
}

func (p *Parser) keyword(kw string) bool {
	return p.current.kind == tokIdent && strings.EqualFold(p.current.val, kw)
}

func (p *Parser) skipKeyword(kw string) bool {
	if p.keyword(kw) {
		p.advance()
		return true
	}
	return false
}

// ParseSelect parses a SELECT statement.
func (p *Parser) ParseSelect() (*SelectStmt, error) {
	if !p.skipKeyword("SELECT") {
		return nil, fmt.Errorf("expected SELECT, got %q", p.current.val)
	}

	stmt := &SelectStmt{Limit: 0}

	// TODO: parse SELECT list (comma-separated expressions with optional AS alias)
	// TODO: parse FROM clause
	// TODO: parse WHERE clause
	// TODO: parse GROUP BY clause
	// TODO: parse ORDER BY clause (with ASC/DESC)
	// TODO: parse LIMIT clause

	return stmt, fmt.Errorf("ParseSelect: not yet implemented — implement the TODO sections above")
}

// ParseExpr parses a WHERE expression (entry point for expression parsing).
// You'll want standard precedence: OR < AND < comparison < NOT.
func (p *Parser) ParseExpr() (Expr, error) {
	return p.parseOr()
}

func (p *Parser) parseOr() (Expr, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.skipKeyword("OR") {
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Left: left, Op: "OR", Right: right}
	}
	return left, nil
}

func (p *Parser) parseAnd() (Expr, error) {
	// TODO: parse left, then loop while AND, parse right
	return p.parseNot()
}

func (p *Parser) parseNot() (Expr, error) {
	if p.skipKeyword("NOT") {
		expr, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Op: "NOT", Expr: expr}, nil
	}
	return p.parseComparison()
}

func (p *Parser) parseComparison() (Expr, error) {
	// TODO: parse primary, then optionally parse comparison operator + primary
	return p.parsePrimary()
}

func (p *Parser) parsePrimary() (Expr, error) {
	// TODO: handle (, literals, function calls, column refs
	switch p.current.kind {
	case tokLParen:
		p.advance()
		expr, err := p.ParseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(tokRParen); err != nil {
			return nil, err
		}
		return expr, nil
	case tokString:
		t := p.advance()
		return &Literal{Value: t.val}, nil
	case tokNumber:
		t := p.advance()
		return &Literal{Value: t.val}, nil // leave as string; executor will coerce
	case tokIdent:
		name := p.current.val
		p.advance()
		// Function call?
		if p.current.kind == tokLParen {
			return p.parseFuncCall(name)
		}
		return &ColumnRef{Name: name}, nil
	case tokStar:
		p.advance()
		return &ColumnRef{Name: "*"}, nil
	}
	return nil, fmt.Errorf("unexpected token %q in expression", p.current.val)
}

func (p *Parser) parseFuncCall(name string) (Expr, error) {
	p.advance() // consume (
	fn := &FuncCallExpr{Name: strings.ToUpper(name)}
	if p.current.kind == tokRParen {
		p.advance()
		return fn, nil
	}
	if p.current.kind == tokStar {
		p.advance()
		fn.Star = true
		_, err := p.expect(tokRParen)
		return fn, err
	}
	for {
		arg, err := p.ParseExpr()
		if err != nil {
			return nil, err
		}
		fn.Args = append(fn.Args, arg)
		if p.current.kind == tokRParen {
			p.advance()
			break
		}
		if _, err := p.expect(tokComma); err != nil {
			return nil, err
		}
	}
	return fn, nil
}

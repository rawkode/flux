// Package parser implements a parser for flux.
//
// The parser is composed of a series of ParseNode objects. The ParseNode has a simple interface.
// It accepts the Scanner which is defined within this package so that alternative scanning
// implementations can be swapped out and different ones can be experimented with.
//
// Each ParseNode takes the Scanner as input. If a ParseNode is able to do something with the next
// token, it will consume the token and produce a new ParseNode that will move to the next stage
// of parsing. For efficiency, a ParseNode may consume more than one token.
//
// Once a ParseNode is not able to consume any more input from the Scanner, it indicates that the
// parsing is in one of two states: terminal or non-terminal. A terminal state means that no more
// input will be accepted no matter what gets fed into the parser. This is most commonly used with
// error states, but it may also indicate that a subtree in the parse tree has finished. A
// non-terminal state means that the system could continue feeding input to the node.
//
// After reaching either a terminal or non-terminal state, the code calling the parser can invoke
// the Get method on the ParseNode. The Get method will construct the ast.Node if it is able to
// and it will return an error if the parser ended in a terminal error state.
//
// The state within each ParseNode is immutable after it has been returned. That means that each stage
// of the parsing could be retained and the state of the Scanner could be preserved for each one
// to be able to replay the entire process. While the state is immutable, the ParseNode's are not
// thread or memory-safe. If you use an older ParseNode, it may use the same memory as a different
// ParseNode for things involving slices and maps. The returned nodes are shallow copies.
//
// TODO(jsternberg): I need to find a way to report the difference between a non-final state error
// and an error-error. Most likely through a custom error type that will both signal that the error
// is recoverable and indicate ways to correct it (such as through completion suggestions maybe?).
package parser

import (
	"fmt"
	"io"
	"strconv"

	"github.com/influxdata/flux/ast"
	"github.com/influxdata/flux/internal/token"
	"github.com/pkg/errors"
)

// Scanner defines the interface for reading a stream of tokens.
type Scanner interface {
	// Scan will scan the next token.
	Scan() (token.Pos, token.Token, string)

	// ScanNoRegex will scan the next token, but exclude any
	// regex literals.
	ScanNoRegex() (token.Pos, token.Token, string)

	// Unread will unread back to the previous location within the Scanner.
	// This can only be called once so the maximum lookahead is one.
	Unread()
}

// ParseNode refers to a portion of the parse tree. It holds the current location
// within the parser so that the parser can be interrupted and continued
// as the controlling program sees is needed. It may also contain information about
// what tokens are expected and which tokens have been encountered to aid with
// syntax highlighting and code completion.
type ParseNode interface {
	// Parse takes the Scanner. Parse will always return a ParseNode. If the ParseNode
	// was able to successfully consume at least one token from the Scanner, then the
	// second argument will be true. If the Scanner could not be advanced, the second
	// argument will be false and the ParseNode will either contain a valid AST fragment,
	// a partial AST fragment, or it will return an error.
	Parse(s Scanner) (ParseNode, bool)

	// Get will retrieve the AST node that has been constructed from the ParseNode.
	Get() (ast.Node, error)
}

// TerminalNode is a (potentially) terminal ParseNode.
type TerminalNode interface {
	ParseNode

	// IsTerminal indicates if this is a terminal node.
	IsTerminal() bool
}

// IsTerminal will determine if a ParseNode is terminal. If the ParseNode does not
// implement the TerminalNode interface, it is assumed to be a non-terminal node.
func IsTerminal(n ParseNode) bool {
	if n, ok := n.(TerminalNode); ok {
		return n.IsTerminal()
	}
	return false
}

// Feed will continuous feed the Scanner to the ParseNode until Parse returns false.
// It will then return the last ParseNode in the series.
func Feed(n ParseNode, s Scanner) ParseNode {
	for {
		var ok bool
		n, ok = n.Parse(s)
		if !ok {
			return n
		}
	}
}

func New() ParseNode {
	return Program{}
}

type Program struct {
	root ast.Program
	stmt ParseNode
}

func (p Program) Parse(s Scanner) (ParseNode, bool) {
	if p.stmt != nil {
		stmt, ok := p.stmt.Parse(s)
		if ok {
			p.stmt = stmt
			return p, true
		}
		// Delay generating the statement temporarily.
		// If we get an EOF, we want to keep the current
		// state before we materialize the object. If
		// we don't get an EOF, then we want to attempt to
		// materialize the statement and add it to the program.
	}

	// Read the next token.
	_, tok, lit := s.Scan()
	if tok != token.EOF && p.stmt != nil {
		// Materialize the previous statement. If we cannot because
		// of an error, then we cannot advance the node and we need
		// to unread the token.
		stmt, err := p.stmt.Get()
		if err != nil {
			s.Unread()
			return p, false
		}

		p.root.Body = append(p.root.Body, stmt.(ast.Statement))
		p.stmt = nil
	}

	switch tok {
	case token.IDENT:
		return p.Ident(lit), true
	case token.EOF:
		return p, false
	default:
		s.Unread()
		return Errorf("unexpected token: %d", tok), false
	}
}

func (p Program) Get() (ast.Node, error) {
	if p.stmt != nil {
		stmt, err := p.stmt.Get()
		if err != nil {
			return nil, err
		}

		stmts := make([]ast.Statement, len(p.root.Body), len(p.root.Body)+1)
		copy(stmts, p.root.Body)
		p.root.Body = append(p.root.Body, stmt.(ast.Statement))
	}
	return &p.root, nil
}

func (p Program) Ident(name string) ParseNode {
	return parseFunc{
		ParseFn: func(s Scanner) (ParseNode, bool) {
			// We are either expecting an assignment or we are in an expression statement.
			switch _, tok, _ := s.Scan(); tok {
			case token.ASSIGN:
				return p.Assignment(name), true
			case token.IDENT:
				// We have a second identifier. If the first identifier was
				// "option", then we have an option statement. Otherwise,
				// there is no valid grammar for two identifiers in a row.
				if name == "option" {
					return Errorf("implement me"), true
				}

				s.Unread()
				return Errorf("invalid token: %d", tok), false
			case token.EOF:
				return Error(io.ErrUnexpectedEOF), false
			default:
				// This is probably an expression statement so read it as if it were one.
				s.Unread()

				p.stmt = ExpressionStatement{
					expr: UnaryExpr{expr: &ast.Identifier{Name: name}},
				}
				return p.Parse(s)
			}
		},
		GetFn: func() (ast.Node, error) {
			return nil, fmt.Errorf("unexpected")
		},
	}
}

func (p Program) Assignment(name string) ParseNode {
	p.stmt = VariableDeclaration{
		name: name,
		expr: UnaryExpr{},
	}
	return p
}

type VariableDeclaration struct {
	name string
	expr ParseNode
}

func (vd VariableDeclaration) Parse(s Scanner) (ParseNode, bool) {
	var ok bool
	vd.expr, ok = vd.expr.Parse(s)
	return vd, ok
}

func (vd VariableDeclaration) Get() (ast.Node, error) {
	expr, err := vd.expr.Get()
	if err != nil {
		return nil, err
	}

	return &ast.VariableDeclaration{
		Declarations: []*ast.VariableDeclarator{
			{
				ID:   &ast.Identifier{Name: vd.name},
				Init: expr.(ast.Expression),
			},
		},
	}, nil
}

type errorNode struct {
	Err error
}

func (e errorNode) Parse(s Scanner) (ParseNode, bool) {
	return e, false
}

func (e errorNode) Get() (ast.Node, error) {
	return nil, e.Err
}

func (e errorNode) IsTerminal() bool {
	return true
}

func Error(err error) ParseNode {
	return errorNode{Err: err}
}

func Errorf(msg string, v ...interface{}) ParseNode {
	return Error(fmt.Errorf(msg, v...))
}

type parseFunc struct {
	ParseFn func(s Scanner) (ParseNode, bool)
	GetFn   func() (ast.Node, error)
}

func (fn parseFunc) Parse(s Scanner) (ParseNode, bool) {
	n, ok := fn.ParseFn(s)
	if !ok {
		if err, ok := n.(errorNode); ok && err.Err == io.ErrUnexpectedEOF {
			return fn, false
		}
	}
	return n, ok
}

func (fn parseFunc) Get() (ast.Node, error) {
	return fn.GetFn()
}

type ExpressionStatement struct {
	expr ParseNode
}

func (e ExpressionStatement) Parse(s Scanner) (ParseNode, bool) {
	var ok bool
	e.expr, ok = e.expr.Parse(s)
	return e, ok
}

func (e ExpressionStatement) Get() (ast.Node, error) {
	expr, err := e.expr.Get()
	if err != nil {
		return nil, err
	}
	return &ast.ExpressionStatement{Expression: expr.(ast.Expression)}, nil
}

type UnaryExpr struct {
	expr ast.Expression
}

func ParseExpr(s Scanner) ParseNode {
	return UnaryExpr{}
}

func (e UnaryExpr) Parse(s Scanner) (ParseNode, bool) {
	if e.expr != nil {
		_, tok, _ := s.ScanNoRegex()
		switch tok {
		case token.DIV:
			return BinaryExpr{
				Expr: ast.BinaryExpression{
					Left:     e.expr,
					Operator: ast.DivisionOperator,
				},
				RHS: UnaryExpr{},
			}, true
		case token.EOF:
			return e, false
		default:
			s.Unread()
			return Errorf("unexpected token: %d", tok), false
		}
	}

	_, tok, lit := s.Scan()
	expr, err := func() (ast.Expression, error) {
		switch tok {
		case token.STRING:
			s, err := strconv.Unquote(lit)
			if err != nil {
				return nil, errors.Wrap(err, "string literal must be surrounded by quotes")
			}
			return &ast.StringLiteral{Value: s}, nil
		case token.INT:
			i, err := strconv.ParseInt(lit, 10, 64)
			if err != nil {
				return nil, errors.Wrap(err, "could not parse integer literal")
			}
			return &ast.IntegerLiteral{Value: i}, nil
		default:
			return nil, fmt.Errorf("unexpected token: %d", tok)
		}
	}()
	if err != nil {
		s.Unread()
		return Error(err), false
	}
	e.expr = expr
	return e, true
}

func (e UnaryExpr) Get() (ast.Node, error) {
	if e.expr == nil {
		return nil, fmt.Errorf("incomplete")
	}
	return e.expr, nil
}

type BinaryExpr struct {
	Expr ast.BinaryExpression
	RHS  ParseNode
}

func (b BinaryExpr) Parse(s Scanner) (ParseNode, bool) {
	var ok bool
	b.RHS, ok = b.RHS.Parse(s)
	return b, ok
}

func (b BinaryExpr) Get() (ast.Node, error) {
	rhs, err := b.RHS.Get()
	if err != nil {
		return nil, err
	}
	b.Expr.Right = rhs.(ast.Expression)
	return &b.Expr, nil
}

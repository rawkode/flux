package parser

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/influxdata/flux/ast"
	"github.com/influxdata/flux/internal/token"
	"github.com/pkg/errors"
)

type Expression struct{}

func (Expression) Parse(s Scanner) (next ParseNode, ok bool) {
	_, tok, lit := s.Scan()
	expr, err := func() (ast.Expression, error) {
		switch tok {
		case token.STRING:
			s, err := strconv.Unquote(lit)
			if err != nil {
				return nil, errors.Wrap(err, "string literal must be surrounded by quotes")
			}
			return &ast.StringLiteral{Value: s}, nil
		case token.REGEX:
			// todo(jsternberg): verify that the regex is surrounded by slashes.
			re, err := regexp.Compile(lit[1 : len(lit)-1])
			if err != nil {
				return nil, errors.Wrap(err, "invalid regular expression")
			}
			return &ast.RegexpLiteral{Value: re}, nil
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
		return Error(err), true
	}
	return UnaryExpr{Expr: expr}, true
}

func (Expression) Get() (ast.Node, error) {
	return nil, fmt.Errorf("expected start of an expression")
}

// UnaryExpr represents a single expression.
type UnaryExpr struct {
	Expr ast.Expression
}

func (e UnaryExpr) Parse(s Scanner) (ParseNode, bool) {
	switch _, tok, _ := s.ScanNoRegex(); tok {
	case token.DIV:
		return BinaryExpr{
			Expr: ast.BinaryExpression{
				Left:     e.Expr,
				Operator: ast.DivisionOperator,
			},
		}, true
	case token.REGEXEQ:
		return BinaryExpr{
			Expr: ast.BinaryExpression{
				Left:     e.Expr,
				Operator: ast.RegexpMatchOperator,
			},
		}, true
	case token.REGEXNEQ:
		return BinaryExpr{
			Expr: ast.BinaryExpression{
				Left:     e.Expr,
				Operator: ast.NotRegexpMatchOperator,
			},
		}, true
	case token.EOF:
		return e, false
	default:
		s.Unread()
		return nil, false
	}
}

func (e UnaryExpr) Get() (ast.Node, error) {
	return e.Expr, nil
}

type BinaryExpr struct {
	Expr ast.BinaryExpression
	RHS  ParseNode
}

func (b BinaryExpr) Parse(s Scanner) (ParseNode, bool) {
	if b.RHS == nil {
		b.RHS = Expression{}
	}
	next, ok := b.RHS.Parse(s)
	if !ok {
		return nil, false
	}
	b.RHS = next
	return b, true
}

func (b BinaryExpr) Get() (ast.Node, error) {
	if b.RHS == nil {
		b.RHS = Expression{}
	}
	rhs, err := b.RHS.Get()
	if err != nil {
		return nil, err
	}
	b.Expr.Right = rhs.(ast.Expression)
	return &b.Expr, nil
}

// Function represents a function call.
type Function struct {
	Name *ast.Identifier
}

func (Function) Parse(s Scanner) (ParseNode, bool) {
	panic("implement me")
}

func (Function) Get() (ast.Node, error) {
	panic("implement me")
}

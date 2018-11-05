package parser

import (
	"fmt"

	"github.com/influxdata/flux/ast"
	"github.com/influxdata/flux/internal/token"
)

type Statement struct{}

func (Statement) Parse(s Scanner) (next ParseNode, ok bool) {
	// Read the next token.
	switch _, tok, lit := s.Scan(); tok {
	case token.IDENT:
		return IdentStatement{
			Identifier: &ast.Identifier{Name: lit},
		}, true
	case token.EOF:
		return nil, false
	default:
		// Likely an expression statement.
		s.Unread()

		stmt := ExpressionStatement{}
		return stmt.Parse(s)
	}
}

func (Statement) Get() (ast.Node, error) {
	return nil, fmt.Errorf("expected start of a statement")
}

type IdentStatement struct {
	Identifier *ast.Identifier
}

func (is IdentStatement) Parse(s Scanner) (next ParseNode, ok bool) {
	// We are either expecting an assignment or we are in an expression statement.
	switch _, tok, _ := s.Scan(); tok {
	case token.ASSIGN:
		return VariableDeclaration{
			LHS: is.Identifier,
		}, true
	case token.IDENT:
		// We have a second identifier. If the first identifier was
		// "option", then we have an option statement. Otherwise,
		// there is no valid grammar for two identifiers in a row.
		if is.Identifier.Name == "option" {
			return Errorf("implement me"), true
		}
		return Errorf("invalid token: %d", tok), true
	case token.EOF:
		return nil, false
	default:
		// This is probably an expression statement so read it as if it were one.
		s.Unread()

		stmt := ExpressionStatement{
			Expr: UnaryExpr{
				Expr: is.Identifier,
			},
		}
		return stmt.Parse(s)
	}
}

func (is IdentStatement) Get() (ast.Node, error) {
	return &ast.ExpressionStatement{
		Expression: is.Identifier,
	}, nil
}

type ExpressionStatement struct {
	Expr ParseNode
}

func (e ExpressionStatement) Parse(s Scanner) (ParseNode, bool) {
	if e.Expr == nil {
		e.Expr = Expression{}
	}
	next, ok := e.Expr.Parse(s)
	if !ok {
		return nil, false
	}
	e.Expr = next
	return e, ok
}

func (e ExpressionStatement) Get() (ast.Node, error) {
	if e.Expr == nil {
		e.Expr = Expression{}
	}
	expr, err := e.Expr.Get()
	if err != nil {
		return nil, err
	}
	return &ast.ExpressionStatement{
		Expression: expr.(ast.Expression),
	}, nil
}

type VariableDeclaration struct {
	LHS *ast.Identifier
	RHS ParseNode
}

func (vd VariableDeclaration) Parse(s Scanner) (ParseNode, bool) {
	if vd.RHS == nil {
		vd.RHS = Expression{}
	}
	next, ok := vd.RHS.Parse(s)
	if !ok {
		return nil, false
	}
	vd.RHS = next
	return vd, true
}

func (vd VariableDeclaration) Get() (ast.Node, error) {
	if vd.RHS == nil {
		vd.RHS = Expression{}
	}
	expr, err := vd.RHS.Get()
	if err != nil {
		return nil, err
	}

	return &ast.VariableDeclaration{
		Declarations: []*ast.VariableDeclarator{
			{
				ID:   vd.LHS,
				Init: expr.(ast.Expression),
			},
		},
	}, nil
}

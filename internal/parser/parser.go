package parser

import (
	"strconv"

	"github.com/influxdata/flux/ast"
	"github.com/influxdata/flux/internal/token"
)

// Scanner defines the interface for reading a stream of tokens.
type Scanner interface {
	// Scan will scan the next token.
	Scan() (pos token.Pos, tok token.Token, lit string)

	// ScanWithRegex will scan the next token and include any regex literals.
	ScanWithRegex() (pos token.Pos, tok token.Token, lit string)

	// Unread will unread back to the previous location within the Scanner.
	// This can only be called once so the maximum lookahead is one.
	Unread()
}

// NewAST parses Flux query and produces an ast.Program.
func NewAST(src Scanner) (*ast.Program, error) {
	p := &parser{s: src}

	var program ast.Program
	for {
		if stmt := p.statement(); stmt != nil {
			program.Body = append(program.Body, stmt)
			continue
		}
		return &program, nil
	}
}

type parser struct {
	s Scanner
}

func (p *parser) statement() ast.Statement {
	switch _, tok, lit := p.s.ScanWithRegex(); tok {
	case token.IDENT:
		ident := &ast.Identifier{Name: lit}
		return p.identStatement(ident)
	case token.RETURN:
		return p.returnStatement()
	case token.LBRACE:
		return p.blockStatement()
	case token.EOF:
		return nil
	default:
		p.s.Unread()
		expr := p.unaryExpr()
		return &ast.ExpressionStatement{
			Expression: expr,
		}
	}
}

func (p *parser) unaryExpr() ast.Expression {
	switch _, tok, lit := p.s.ScanWithRegex(); tok {
	case token.IDENT:
		return &ast.Identifier{Name: lit}
	case token.INT:
		n, err := strconv.ParseInt(lit, 10, 64)
		if err != nil {
			panic(err)
		}
		return &ast.IntegerLiteral{Value: n}
	case token.FLOAT:
		n, err := strconv.ParseFloat(lit, 64)
		if err != nil {
			panic(err)
		}
		return &ast.FloatLiteral{Value: n}
	default:
		panic("implement me")
	}
}

func (p *parser) identStatement(ident *ast.Identifier) ast.Statement {
	switch _, tok, lit := p.s.Scan(); tok {
	case token.IDENT:
		if ident.Name == "option" {
			return p.option(lit)
		}
		panic("implement me")
	case token.ASSIGN:
		expr := p.unaryExpr()
		return &ast.VariableDeclaration{
			Declarations: []*ast.VariableDeclarator{
				{
					ID:   ident,
					Init: expr,
				},
			},
		}
	default:
		p.s.Unread()
		return &ast.ExpressionStatement{
			Expression: p.binaryExpr(ident),
		}
	}
}

func (p *parser) binaryExpr(expr ast.Expression) ast.Expression {
	// TODO(jsternberg): Implement binary operators.
	switch _, tok, _ := p.s.Scan(); tok {
	case token.LPAREN:
		return p.callExpr(expr)
	default:
		p.s.Unread()
		return expr
	}
}

func (p *parser) callExpr(callee ast.Expression) ast.Expression {
	switch _, tok, _ := p.s.Scan(); tok {
	case token.IDENT:
		panic("implement me")
	case token.RPAREN:
		return &ast.CallExpression{
			Callee: callee,
		}
	default:
		panic("implement me")
	}
}

func (p *parser) option(name string) ast.Statement {
	panic("implement me")
}

func (p *parser) returnStatement() ast.Statement {
	panic("implement me")
}

func (p *parser) blockStatement() ast.Statement {
	panic("implement me")
}

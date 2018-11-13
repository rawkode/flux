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
		expr := p.expression()
		return &ast.ExpressionStatement{
			Expression: expr,
		}
	}
}

func (p *parser) expression() ast.Expression {
	expr := p.logicalExpression(p.primary())
	if expr == nil {
		return nil
	}
	return p.pipeExpression(expr)
}

func (p *parser) pipeExpression(expr ast.Expression) ast.Expression {
	for {
		switch _, tok, _ := p.s.Scan(); tok {
		case token.PIPE:
			rhs := p.logicalExpression(p.primary())
			call, ok := rhs.(*ast.CallExpression)
			if !ok {
				// todo(jsternberg): the pipe requires the second
				// argument to be a call expression because the peg
				// parser wasn't capable of anything else. until we
				// fix the ast to accept any expression, this has to
				// be the case.
				panic("implement me")
			}
			expr = &ast.PipeExpression{
				Argument: expr,
				Call:     call,
			}
		default:
			p.s.Unread()
			return p.logicalExpression(expr)
		}
	}
}

func (p *parser) logicalExpression(expr ast.Expression) ast.Expression {
	return p.callExpr(expr)
}

func (p *parser) callExpr(callee ast.Expression) ast.Expression {
	if callee == nil {
		return nil
	}
	switch _, tok, _ := p.s.ScanWithRegex(); tok {
	case token.LPAREN:
		args := p.expressionList(token.RPAREN)
		return &ast.CallExpression{
			Callee:    callee,
			Arguments: args,
		}
	default:
		p.s.Unread()
		return callee
	}
}

func (p *parser) expressionList(until token.Token) []ast.Expression {
	switch _, tok, _ := p.s.ScanWithRegex(); tok {
	case until:
		return nil
	default:
		panic("implement me")
	}
}

func (p *parser) primary() ast.Expression {
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
		p.s.Unread()
		return nil
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
		expr := p.expression()
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
		expr := p.logicalExpression(ident)
		return &ast.ExpressionStatement{
			Expression: p.pipeExpression(expr),
		}
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

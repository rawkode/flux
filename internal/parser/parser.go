package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

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
func NewAST(src Scanner) *ast.Program {
	p := &parser{
		s: &scannerSkipComments{
			Scanner: src,
		},
	}
	return p.program()
}

// scannerSkipComments is a temporary Scanner used for stripping comments
// from the input stream. We want to attach comments to nodes within the
// AST, but first we want to have feature parity with the old parser so
// the easiest method is just to strip comments at the moment.
type scannerSkipComments struct {
	Scanner
}

func (s *scannerSkipComments) Scan() (pos token.Pos, tok token.Token, lit string) {
	for {
		pos, tok, lit = s.Scanner.Scan()
		if tok != token.COMMENT {
			return pos, tok, lit
		}
	}
}

func (s *scannerSkipComments) ScanWithRegex() (pos token.Pos, tok token.Token, lit string) {
	for {
		pos, tok, lit = s.Scanner.ScanWithRegex()
		if tok != token.COMMENT {
			return pos, tok, lit
		}
	}
}

type parser struct {
	s Scanner
}

func (p *parser) program() *ast.Program {
	program := &ast.Program{}
	program.Body = p.statementList(token.EOF)
	return program
}

func (p *parser) statementList(eof token.Token) []ast.Statement {
	var stmts []ast.Statement
	for {
		stmt := p.statement(eof)
		if stmt == nil {
			return stmts
		}
		stmts = append(stmts, stmt)
	}
}

func (p *parser) statement(eof token.Token) ast.Statement {
	switch pos, tok, lit := p.s.ScanWithRegex(); tok {
	case token.IDENT:
		ident := &ast.Identifier{Name: lit}
		return p.identStatement(ident)
	case token.INT, token.FLOAT, token.STRING, token.REGEX,
		token.DURATION, token.LPAREN, token.LBRACK, token.LBRACE,
		token.ADD, token.SUB, token.NOT:
		lhs := p.unaryExprEval(pos, tok, lit)
		return p.exprStatement(lhs)
	case token.ILLEGAL:
		return p.statement(eof)
	case token.RETURN:
		expr := p.expression()
		return &ast.ReturnStatement{Argument: expr}
	case eof, token.EOF:
		return nil
	default:
		return p.statement(eof)
	}
}

func (p *parser) identStatement(ident *ast.Identifier) ast.Statement {
	switch _, tok, lit := p.s.Scan(); tok {
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
	case token.IDENT:
		// This might be an option statement.
		if ident.Name == "option" {
			return p.optionStatement(&ast.Identifier{Name: lit})
		}
		// Otherwise fallthrough to be handled like an expression (invalid).
		fallthrough
	default:
		p.s.Unread()
		return p.exprStatement(ident)
	}
}

func (p *parser) optionStatement(name *ast.Identifier) ast.Statement {
	p.expect(p.s.Scan, token.ASSIGN)
	expr := p.expression()
	return &ast.OptionStatement{
		Declaration: &ast.VariableDeclarator{
			ID:   name,
			Init: expr,
		},
	}
}

func (p *parser) blockStatement() ast.Statement {
	stmts := p.statementList(token.RBRACE)
	return &ast.BlockStatement{Body: stmts}
}

func (p *parser) exprStatement(lhs ast.Expression) ast.Statement {
	expr := p.exprStart(lhs)
	return &ast.ExpressionStatement{
		Expression: expr,
	}
}

func (p *parser) expression() ast.Expression {
	lhs := p.unaryExpr()
	return p.exprStart(lhs)
}

func (p *parser) expressionList(until token.Token) []ast.Expression {
	var exprs []ast.Expression
NEXT:
	for {
		pos, tok, lit := p.s.ScanWithRegex()
		switch tok {
		case until:
			return exprs
		default:
			lhs := p.unaryExprEval(pos, tok, lit)
			expr := p.exprStart(lhs)
			exprs = append(exprs, expr)
		}

		// Search for a comma or the until token.
		for {
			switch _, tok, _ := p.s.Scan(); tok {
			case until:
				return exprs
			case token.COMMA:
				continue NEXT
			case token.EOF:
				return exprs
			default:
				// todo(jsternberg): handle invalid tokens.
			}
		}
	}
}

func (p *parser) propertyList(kvs token.Token, until token.Token) []*ast.Property {
	var properties []*ast.Property
START:
	for {
		_, tok, ident := p.s.ScanWithRegex()
		switch tok {
		case token.IDENT:
			// Nothing to do. We found the identifier.
		case until:
			return properties
		default:
			// todo(jsternberg): create an error property and rescan.
			continue
		}

		// Look for a colon to separate.
		// todo(jsternberg): Figure out how to construct the AST for
		// these types of errors.
		for {
			if _, tok, _ := p.s.Scan(); tok == kvs || tok == token.EOF {
				break
			} else if tok == until {
				// No value assigned to this property.
				properties = append(properties, &ast.Property{
					Key: &ast.Identifier{Name: ident},
				})
				return properties
			} else if tok == token.COMMA {
				// No value assigned to this property.
				properties = append(properties, &ast.Property{
					Key: &ast.Identifier{Name: ident},
				})
				continue START
			}
		}

		expr := p.expression()
		properties = append(properties, &ast.Property{
			Key:   &ast.Identifier{Name: ident},
			Value: expr,
		})
		// todo(jsternberg): determine how to put errors
		// in the ast here. The issue is we know what we need
		// and we should continue searching for what we need to
		// get the most accurate AST, but how we represent that
		// in the AST is an open question.
		for {
			switch _, tok, _ := p.s.Scan(); tok {
			case token.COMMA:
			case until:
				return properties
			default:
				continue
			}
			break
		}
	}
}

func (p *parser) exprStart(lhs ast.Expression) ast.Expression {
	for {
		_, tok, _ := p.s.Scan()
		if ok := p.handleLogicalExpr(&lhs, tok); !ok {
			p.s.Unread()
			return lhs
		}
	}
}

func (p *parser) handleLogicalExpr(lhs *ast.Expression, tok token.Token) bool {
	switch tok {
	case token.AND:
		*lhs = p.logicalExpr(*lhs, ast.AndOperator)
		return true
	case token.OR:
		*lhs = p.logicalExpr(*lhs, ast.OrOperator)
		return true
	default:
		return p.handleComparisonExpr(lhs, tok)
	}
}

func (p *parser) logicalExpr(lhs ast.Expression, op ast.LogicalOperatorKind) ast.Expression {
	rhs := p.unaryExpr()
	for {
		_, tok, _ := p.s.Scan()
		if ok := p.handleComparisonExpr(&rhs, tok); !ok {
			p.s.Unread()
			return &ast.LogicalExpression{
				Operator: op,
				Left:     lhs,
				Right:    rhs,
			}
		}
	}
}

func (p *parser) handleComparisonExpr(lhs *ast.Expression, tok token.Token) bool {
	switch tok {
	case token.EQ:
		*lhs = p.comparisonExpr(*lhs, ast.EqualOperator)
		return true
	case token.NEQ:
		*lhs = p.comparisonExpr(*lhs, ast.NotEqualOperator)
		return true
	case token.REGEXEQ:
		*lhs = p.comparisonExpr(*lhs, ast.RegexpMatchOperator)
		return true
	case token.REGEXNEQ:
		*lhs = p.comparisonExpr(*lhs, ast.NotRegexpMatchOperator)
		return true
	default:
		return p.handleMultiplicativeExpr(lhs, tok)
	}
}

func (p *parser) comparisonExpr(lhs ast.Expression, op ast.OperatorKind) ast.Expression {
	rhs := p.unaryExpr()
	for {
		_, tok, _ := p.s.Scan()
		if ok := p.handleMultiplicativeExpr(&rhs, tok); !ok {
			p.s.Unread()
			return &ast.BinaryExpression{
				Operator: op,
				Left:     lhs,
				Right:    rhs,
			}
		}
	}
}

func (p *parser) handleMultiplicativeExpr(lhs *ast.Expression, tok token.Token) bool {
	switch tok {
	case token.MUL:
		*lhs = p.multiplicativeExpr(*lhs, ast.MultiplicationOperator)
		return true
	case token.DIV:
		*lhs = p.multiplicativeExpr(*lhs, ast.DivisionOperator)
		return true
	default:
		return p.handleAdditiveExpr(lhs, tok)
	}
}

func (p *parser) multiplicativeExpr(lhs ast.Expression, op ast.OperatorKind) ast.Expression {
	rhs := p.unaryExpr()
	for {
		_, tok, _ := p.s.Scan()
		if ok := p.handleAdditiveExpr(&rhs, tok); !ok {
			p.s.Unread()
			return &ast.BinaryExpression{
				Operator: op,
				Left:     lhs,
				Right:    rhs,
			}
		}
	}
}

func (p *parser) handleAdditiveExpr(lhs *ast.Expression, tok token.Token) bool {
	switch tok {
	case token.ADD:
		*lhs = p.additiveExpr(*lhs, ast.AdditionOperator)
		return true
	case token.SUB:
		*lhs = p.additiveExpr(*lhs, ast.SubtractionOperator)
		return true
	default:
		return p.handlePipeExpr(lhs, tok)
	}
}

func (p *parser) additiveExpr(lhs ast.Expression, op ast.OperatorKind) ast.Expression {
	rhs := p.unaryExpr()
	for {
		_, tok, _ := p.s.Scan()
		if ok := p.handlePipeExpr(&rhs, tok); !ok {
			p.s.Unread()
			return &ast.BinaryExpression{
				Operator: op,
				Left:     lhs,
				Right:    rhs,
			}
		}
	}
}

func (p *parser) handlePipeExpr(lhs *ast.Expression, tok token.Token) bool {
	switch tok {
	case token.PIPE_FORWARD:
		*lhs = p.pipeExpr(*lhs)
		return true
	default:
		return p.handlePostfixExpr(lhs, tok)
	}
}

func (p *parser) pipeExpr(lhs ast.Expression) ast.Expression {
	rhs := p.unaryExpr()
	for {
		_, tok, _ := p.s.Scan()
		if ok := p.handlePostfixExpr(&rhs, tok); !ok {
			p.s.Unread()
			return &ast.PipeExpression{
				Argument: lhs,
				Call:     rhs.(*ast.CallExpression),
			}
		}
	}
}

func (p *parser) handlePostfixExpr(lhs *ast.Expression, tok token.Token) bool {
	switch tok {
	case token.LPAREN:
		*lhs = p.callExpr(*lhs)
		return true
	case token.DOT:
		*lhs = p.dotExpr(*lhs)
		return true
	case token.LBRACK:
		*lhs = p.indexExpr(*lhs)
		return true
	default:
		return false
	}
}

func (p *parser) callExpr(callee ast.Expression) ast.Expression {
	if params := p.propertyList(token.COLON, token.RPAREN); len(params) > 0 {
		return &ast.CallExpression{
			Callee: callee,
			Arguments: []ast.Expression{
				&ast.ObjectExpression{Properties: params},
			},
		}
	}
	return &ast.CallExpression{Callee: callee}
}

func (p *parser) dotExpr(callee ast.Expression) ast.Expression {
	for {
		switch _, tok, lit := p.s.ScanWithRegex(); tok {
		case token.IDENT:
			return &ast.MemberExpression{
				Object:   callee,
				Property: &ast.Identifier{Name: lit},
			}
		case token.EOF:
			return nil
		}
	}
}

func (p *parser) indexExpr(callee ast.Expression) ast.Expression {
	expr := p.expression()
	// todo(jsternberg): do something about the wrong token here.
	p.expect(p.s.Scan, token.RBRACK)

	if lit, ok := expr.(*ast.StringLiteral); ok {
		return &ast.MemberExpression{
			Object:   callee,
			Property: lit,
		}
	}
	return &ast.IndexExpression{
		Array: callee,
		Index: expr,
	}
}

func (p *parser) unaryExpr() ast.Expression {
	return p.unaryExprEval(p.s.ScanWithRegex())
}

func (p *parser) unaryExprEval(pos token.Pos, tok token.Token, lit string) ast.Expression {
	switch tok {
	case token.ADD:
		return &ast.UnaryExpression{
			Operator: ast.AdditionOperator,
			Argument: p.primaryExpr(p.s.ScanWithRegex()),
		}
	case token.SUB:
		return &ast.UnaryExpression{
			Operator: ast.SubtractionOperator,
			Argument: p.primaryExpr(p.s.ScanWithRegex()),
		}
	case token.NOT:
		return &ast.UnaryExpression{
			Operator: ast.NotOperator,
			Argument: p.primaryExpr(p.s.ScanWithRegex()),
		}
	default:
		return p.primaryExpr(pos, tok, lit)
	}
}

func (p *parser) primaryExpr(pos token.Pos, tok token.Token, lit string) ast.Expression {
	switch tok {
	case token.IDENT:
		return &ast.Identifier{Name: lit}
	case token.INT:
		value, err := strconv.ParseInt(lit, 10, 64)
		if err != nil {
			panic(err)
		}
		return &ast.IntegerLiteral{Value: value}
	case token.FLOAT:
		value, err := strconv.ParseFloat(lit, 64)
		if err != nil {
			panic(err)
		}
		return &ast.FloatLiteral{Value: value}
	case token.STRING:
		value, err := strconv.Unquote(lit)
		if err != nil {
			panic(err)
		}
		return &ast.StringLiteral{Value: value}
	case token.REGEX:
		value, err := parseRegexp(lit)
		if err != nil {
			panic(err)
		}
		return &ast.RegexpLiteral{Value: value}
	case token.DURATION:
		values, err := parseDuration(lit)
		if err != nil {
			panic(err)
		}
		return &ast.DurationLiteral{Values: values}
	case token.LBRACK:
		exprs := p.expressionList(token.RBRACK)
		return &ast.ArrayExpression{Elements: exprs}
	case token.LBRACE:
		properties := p.propertyList(token.COLON, token.RBRACE)
		return &ast.ObjectExpression{Properties: properties}
	case token.LPAREN:
		return p.parenExpr()
	default:
		panic("invalid expression")
	}
}

func (p *parser) parenExpr() ast.Expression {
	// When we see an open parenthesis, this could either be a normal
	// expression or it might be an arrow expression.
	_, tok, lit := p.s.ScanWithRegex()
	switch tok {
	case token.RPAREN:
		return p.arrowExpr(nil)
	case token.IDENT:
		// This could be a normal parenthesis expression or it
		// could be a function call.
		lhs := &ast.Identifier{Name: lit}
		switch _, tok, _ := p.s.Scan(); tok {
		case token.RPAREN:
			// This could be an identifier in parenthesis or it could
			// be a function call.
			if _, tok, _ := p.s.Scan(); tok == token.ARROW {
				return p.arrowExprBody([]*ast.Property{
					{Key: lhs},
				})
			}
			p.s.Unread()
			return lhs
		case token.ASSIGN:
			value := p.expression()
			switch _, tok, _ := p.s.Scan(); tok {
			case token.COMMA:
				// We are reading more function parameters. This
				// may be empty and it's fine.
				rest := p.propertyList(token.ASSIGN, token.RPAREN)
				properties := make([]*ast.Property, 0, len(rest)+1)
				properties = append(properties, &ast.Property{
					Key:   lhs,
					Value: value,
				})
				properties = append(properties, rest...)
				return p.arrowExpr(properties)
			default:
				p.expect(p.s.Scan, token.RPAREN)
				fallthrough
			case token.RPAREN:
				// One parameter with a value.
				return p.arrowExpr([]*ast.Property{
					{
						Key:   lhs,
						Value: value,
					},
				})
			}
		case token.COMMA:
			// We are reading more function parameters. This
			// may be empty and it's fine.
			rest := p.propertyList(token.ASSIGN, token.RPAREN)
			properties := make([]*ast.Property, 0, len(rest)+1)
			properties = append(properties, &ast.Property{
				Key: lhs,
			})
			properties = append(properties, rest...)
			return p.arrowExpr(properties)
		default:
			p.s.Unread()
			expr := p.exprStart(lhs)
			p.expect(p.s.Scan, token.RPAREN)
			return expr
		}
	default:
		// Unread the token as this has to be an expression.
		p.s.Unread()
		expr := p.expression()
		p.expect(p.s.Scan, token.RPAREN)
		return expr
	}
}

func (p *parser) arrowExpr(params []*ast.Property) ast.Expression {
	// The parenthesis closes immediately. If we see this, it
	// signals we are entering an arrow expression.
	// todo(jsternberg): consider a better error message here
	// if it isn't an arrow.
	p.expect(p.s.Scan, token.ARROW)
	return p.arrowExprBody(params)
}

func (p *parser) arrowExprBody(params []*ast.Property) ast.Expression {
	pos, tok, lit := p.s.ScanWithRegex()
	return &ast.ArrowFunctionExpression{
		Params: params,
		Body: func() ast.Node {
			switch tok {
			case token.LBRACE:
				return p.blockStatement()
			default:
				lhs := p.unaryExprEval(pos, tok, lit)
				return p.exprStart(lhs)
			}
		}(),
	}
}

// expect is a temporary method for when we are expecting a certain token.
// It skips past every other token until we find the correct one. In the future,
// we need to define how these become errors.
func (p *parser) expect(scanMethod func() (token.Pos, token.Token, string), tokens ...token.Token) {
	for {
		_, tok, _ := scanMethod()
		if tok == token.EOF {
			return
		}
		for _, etok := range tokens {
			if tok == etok {
				return
			}
		}
	}
}

func parseDuration(lit string) ([]ast.Duration, error) {
	var values []ast.Duration
	for len(lit) > 0 {
		n := 0
		for n < len(lit) {
			ch, size := utf8.DecodeRuneInString(lit[n:])
			if size == 0 {
				panic("invalid rune in duration")
			}

			if !unicode.IsDigit(ch) {
				break
			}
			n += size
		}

		magnitude, err := strconv.ParseInt(lit[:n], 10, 64)
		if err != nil {
			return nil, err
		}
		lit = lit[n:]

		n = 0
		for n < len(lit) {
			ch, size := utf8.DecodeRuneInString(lit[n:])
			if size == 0 {
				panic("invalid rune in duration")
			}

			if !unicode.IsLetter(ch) {
				break
			}
			n += size
		}
		unit := lit[:n]
		if unit == "µs" {
			unit = "us"
		}
		values = append(values, ast.Duration{
			Magnitude: magnitude,
			Unit:      unit,
		})
		lit = lit[n:]
	}
	return values, nil
}

func parseRegexp(lit string) (*regexp.Regexp, error) {
	if len(lit) < 3 {
		return nil, fmt.Errorf("regexp must be at least 3 characters")
	}

	if lit[0] != '/' {
		return nil, fmt.Errorf("regexp literal must start with a slash")
	} else if lit[len(lit)-1] != '/' {
		return nil, fmt.Errorf("regexp literal must end with a slash")
	}

	expr := lit[1 : len(lit)-1]
	if index := strings.Index(expr, "\\/"); index != -1 {
		expr = strings.Replace(expr, "\\/", "/", -1)
	}
	return regexp.Compile(expr)
}

func tokstr(tok token.Token, lit string) string {
	return lit
}

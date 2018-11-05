package parser_test

import (
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/influxdata/flux/ast"
	"github.com/influxdata/flux/internal/parser"
	"github.com/influxdata/flux/internal/token"
)

var CompareOptions = []cmp.Option{
	cmp.Transformer("", func(re *regexp.Regexp) string {
		return re.String()
	}),
}

type Token struct {
	Pos   token.Pos
	Token token.Token
	Lit   string
}

type Scanner struct {
	Tokens   []Token
	i        int
	buffered bool
}

func (s *Scanner) Scan() (token.Pos, token.Token, string) {
	if s.i >= len(s.Tokens) {
		return 0, token.EOF, ""
	}
	tok := s.Tokens[s.i]
	s.i++
	s.buffered = false
	return tok.Pos, tok.Token, tok.Lit
}

func (s *Scanner) ScanNoRegex() (token.Pos, token.Token, string) {
	pos, tok, lit := s.Scan()
	if tok == token.REGEX {
		// The parser was asking for a non regex token and our static
		// scanner gave it one. This indicates a bug in the parser since
		// the parser should have invoked Scan instead.
		s.Unread()
		return 0, token.ILLEGAL, ""
	}
	return pos, tok, lit
}

func (s *Scanner) Unread() {
	// Buffered indicates that the value is "buffered". Since we keep everything
	// in memory, we use it to prevent unread from going backwards more than once
	// to prevent accidentally using a lookahead of 2 when testing the parser.
	if !s.buffered {
		s.buffered = true
		s.i--
	}
}

func TestParser(t *testing.T) {
	for _, tt := range []struct {
		Name   string
		Tokens []Token
		Result *ast.Program
	}{
		{
			Name: "variable declaration",
			Tokens: []Token{
				{Token: token.IDENT, Lit: "a"},
				{Token: token.ASSIGN, Lit: "="},
				{Token: token.STRING, Lit: `"hello"`},
			},
			Result: &ast.Program{
				Body: []ast.Statement{
					&ast.VariableDeclaration{
						Declarations: []*ast.VariableDeclarator{
							{ID: &ast.Identifier{Name: "a"}, Init: &ast.StringLiteral{Value: "hello"}},
						},
					},
				},
			},
		},
		{
			Name: "binary expression - division",
			Tokens: []Token{
				{Token: token.IDENT, Lit: "a"},
				{Token: token.DIV, Lit: "/"},
				{Token: token.INT, Lit: "2"},
			},
			Result: &ast.Program{
				Body: []ast.Statement{
					&ast.ExpressionStatement{
						Expression: &ast.BinaryExpression{
							Operator: ast.DivisionOperator,
							Left:     &ast.Identifier{Name: "a"},
							Right:    &ast.IntegerLiteral{Value: 2},
						},
					},
				},
			},
		},
		{
			Name: "binary expression - regex match",
			Tokens: []Token{
				{Token: token.REGEX, Lit: "/a/"},
				{Token: token.REGEXEQ, Lit: "=~"},
				{Token: token.STRING, Lit: `"abc"`},
			},
			Result: &ast.Program{
				Body: []ast.Statement{
					&ast.ExpressionStatement{
						Expression: &ast.BinaryExpression{
							Operator: ast.RegexpMatchOperator,
							Left:     &ast.RegexpLiteral{Value: regexp.MustCompile(`a`)},
							Right:    &ast.StringLiteral{Value: "abc"},
						},
					},
				},
			},
		},
		{
			Name: "binary expression - regex not match",
			Tokens: []Token{
				{Token: token.REGEX, Lit: "/a/"},
				{Token: token.REGEXNEQ, Lit: "!~"},
				{Token: token.STRING, Lit: `"abc"`},
			},
			Result: &ast.Program{
				Body: []ast.Statement{
					&ast.ExpressionStatement{
						Expression: &ast.BinaryExpression{
							Operator: ast.NotRegexpMatchOperator,
							Left:     &ast.RegexpLiteral{Value: regexp.MustCompile(`a`)},
							Right:    &ast.StringLiteral{Value: "abc"},
						},
					},
				},
			},
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			scanner := &Scanner{Tokens: tt.Tokens}
			n := parser.Feed(parser.Program{}, scanner)
			result, err := n.Get()
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if got, want := result, tt.Result; !cmp.Equal(want, got, CompareOptions...) {
				t.Fatalf("unexpected statement -want/+got\n%s", cmp.Diff(want, got, CompareOptions...))
			}
		})
	}
}

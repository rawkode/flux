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
	pos, tok, lit := s.ScanWithRegex()
	if tok == token.REGEX {
		// The parser was asking for a non regex token and our static
		// scanner gave it one. This indicates a bug in the parser since
		// the parser should have invoked Scan instead.
		s.Unread()
		return 0, token.ILLEGAL, ""
	}
	return pos, tok, lit
}

func (s *Scanner) ScanWithRegex() (token.Pos, token.Token, string) {
	if s.i >= len(s.Tokens) {
		s.buffered = true
		return 0, token.EOF, ""
	}
	tok := s.Tokens[s.i]
	s.i++
	s.buffered = false
	return tok.Pos, tok.Token, tok.Lit
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
		name   string
		tokens []Token
		want   *ast.Program
		skip   bool
	}{
		{
			name: "from",
			tokens: []Token{
				{Token: token.IDENT, Lit: `from`},
				{Token: token.LPAREN, Lit: `(`},
				{Token: token.RPAREN, Lit: `)`},
			},
			want: &ast.Program{
				Body: []ast.Statement{
					&ast.ExpressionStatement{
						Expression: &ast.CallExpression{
							Callee: &ast.Identifier{
								Name: "from",
							},
						},
					},
				},
			},
		},
		{
			name: "identifier with number",
			tokens: []Token{
				{Token: token.IDENT, Lit: `tan2`},
				{Token: token.LPAREN, Lit: `(`},
				{Token: token.RPAREN, Lit: `)`},
			},
			want: &ast.Program{
				Body: []ast.Statement{
					&ast.ExpressionStatement{
						Expression: &ast.CallExpression{
							Callee: &ast.Identifier{
								Name: "tan2",
							},
						},
					},
				},
			},
		},
		{
			name: "declare variable as an int",
			tokens: []Token{
				{Token: token.IDENT, Lit: `howdy`},
				{Token: token.ASSIGN, Lit: `=`},
				{Token: token.INT, Lit: `1`},
			},
			want: &ast.Program{
				Body: []ast.Statement{
					&ast.VariableDeclaration{
						Declarations: []*ast.VariableDeclarator{{
							ID:   &ast.Identifier{Name: "howdy"},
							Init: &ast.IntegerLiteral{Value: 1},
						}},
					},
				},
			},
		},
		{
			name: "declare variable as a float",
			tokens: []Token{
				{Token: token.IDENT, Lit: `howdy`},
				{Token: token.ASSIGN, Lit: `=`},
				{Token: token.FLOAT, Lit: `1.1`},
			},
			want: &ast.Program{
				Body: []ast.Statement{
					&ast.VariableDeclaration{
						Declarations: []*ast.VariableDeclarator{{
							ID:   &ast.Identifier{Name: "howdy"},
							Init: &ast.FloatLiteral{Value: 1.1},
						}},
					},
				},
			},
		},
		{
			name: "use variable to declare something",
			tokens: []Token{
				{Token: token.IDENT, Lit: `howdy`},
				{Token: token.ASSIGN, Lit: `=`},
				{Token: token.INT, Lit: `1`},
				{Token: token.IDENT, Lit: `from`},
				{Token: token.LPAREN, Lit: `(`},
				{Token: token.RPAREN, Lit: `)`},
			},
			want: &ast.Program{
				Body: []ast.Statement{
					&ast.VariableDeclaration{
						Declarations: []*ast.VariableDeclarator{{
							ID:   &ast.Identifier{Name: "howdy"},
							Init: &ast.IntegerLiteral{Value: 1},
						}},
					},
					&ast.ExpressionStatement{
						Expression: &ast.CallExpression{
							Callee: &ast.Identifier{
								Name: "from",
							},
						},
					},
				},
			},
		},
		{
			name: "pipe expression",
			tokens: []Token{
				{Token: token.IDENT, Lit: `from`},
				{Token: token.LPAREN, Lit: `(`},
				{Token: token.RPAREN, Lit: `)`},
				{Token: token.PIPE, Lit: `|>`},
				{Token: token.IDENT, Lit: `count`},
				{Token: token.LPAREN, Lit: `(`},
				{Token: token.RPAREN, Lit: `)`},
			},
			want: &ast.Program{
				Body: []ast.Statement{
					&ast.ExpressionStatement{
						Expression: &ast.PipeExpression{
							Argument: &ast.CallExpression{
								Callee:    &ast.Identifier{Name: "from"},
								Arguments: nil,
							},
							Call: &ast.CallExpression{
								Callee:    &ast.Identifier{Name: "count"},
								Arguments: nil,
							},
						},
					},
				},
			},
		},
		{
			name: "literal pipe expression",
			tokens: []Token{
				{Token: token.INT, Lit: `5`},
				{Token: token.PIPE, Lit: `|>`},
				{Token: token.IDENT, Lit: `pow2`},
				{Token: token.LPAREN, Lit: `(`},
				{Token: token.RPAREN, Lit: `)`},
			},
			want: &ast.Program{
				Body: []ast.Statement{
					&ast.ExpressionStatement{
						Expression: &ast.PipeExpression{
							Argument: &ast.IntegerLiteral{Value: 5},
							Call: &ast.CallExpression{
								Callee:    &ast.Identifier{Name: "pow2"},
								Arguments: nil,
							},
						},
					},
				},
			},
		},
		{
			name: "multiple pipe expressions",
			tokens: []Token{
				{Token: token.IDENT, Lit: `from`},
				{Token: token.LPAREN, Lit: `(`},
				{Token: token.RPAREN, Lit: `)`},
				{Token: token.PIPE, Lit: `|>`},
				{Token: token.IDENT, Lit: `range`},
				{Token: token.LPAREN, Lit: `(`},
				{Token: token.RPAREN, Lit: `)`},
				{Token: token.PIPE, Lit: `|>`},
				{Token: token.IDENT, Lit: `filter`},
				{Token: token.LPAREN, Lit: `(`},
				{Token: token.RPAREN, Lit: `)`},
				{Token: token.PIPE, Lit: `|>`},
				{Token: token.IDENT, Lit: `count`},
				{Token: token.LPAREN, Lit: `(`},
				{Token: token.RPAREN, Lit: `)`},
			},
			want: &ast.Program{
				Body: []ast.Statement{
					&ast.ExpressionStatement{
						Expression: &ast.PipeExpression{
							Argument: &ast.PipeExpression{
								Argument: &ast.PipeExpression{
									Argument: &ast.CallExpression{
										Callee: &ast.Identifier{Name: "from"},
									},
									Call: &ast.CallExpression{
										Callee: &ast.Identifier{Name: "range"},
									},
								},
								Call: &ast.CallExpression{
									Callee: &ast.Identifier{Name: "filter"},
								},
							},
							Call: &ast.CallExpression{
								Callee: &ast.Identifier{Name: "count"},
							},
						},
					},
				},
			},
		},
		{
			name: "two variables for two froms",
			tokens: []Token{
				{Token: token.IDENT, Lit: `howdy`},
				{Token: token.ASSIGN, Lit: `=`},
				{Token: token.IDENT, Lit: `from`},
				{Token: token.LPAREN, Lit: `(`},
				{Token: token.RPAREN, Lit: `)`},
				{Token: token.IDENT, Lit: `doody`},
				{Token: token.ASSIGN, Lit: `=`},
				{Token: token.IDENT, Lit: `from`},
				{Token: token.LPAREN, Lit: `(`},
				{Token: token.RPAREN, Lit: `)`},
				{Token: token.IDENT, Lit: `howdy`},
				{Token: token.PIPE, Lit: `|>`},
				{Token: token.IDENT, Lit: `count`},
				{Token: token.LPAREN, Lit: `(`},
				{Token: token.RPAREN, Lit: `)`},
				{Token: token.IDENT, Lit: `doody`},
				{Token: token.PIPE, Lit: `|>`},
				{Token: token.IDENT, Lit: `sum`},
				{Token: token.LPAREN, Lit: `(`},
				{Token: token.RPAREN, Lit: `)`},
			},
			want: &ast.Program{
				Body: []ast.Statement{
					&ast.VariableDeclaration{
						Declarations: []*ast.VariableDeclarator{{
							ID: &ast.Identifier{
								Name: "howdy",
							},
							Init: &ast.CallExpression{
								Callee: &ast.Identifier{
									Name: "from",
								},
							},
						}},
					},
					&ast.VariableDeclaration{
						Declarations: []*ast.VariableDeclarator{{
							ID: &ast.Identifier{
								Name: "doody",
							},
							Init: &ast.CallExpression{
								Callee: &ast.Identifier{
									Name: "from",
								},
							},
						}},
					},
					&ast.ExpressionStatement{
						Expression: &ast.PipeExpression{
							Argument: &ast.Identifier{Name: "howdy"},
							Call: &ast.CallExpression{
								Callee: &ast.Identifier{
									Name: "count",
								},
							},
						},
					},
					&ast.ExpressionStatement{
						Expression: &ast.PipeExpression{
							Argument: &ast.Identifier{Name: "doody"},
							Call: &ast.CallExpression{
								Callee: &ast.Identifier{
									Name: "sum",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "var as binary expression of other vars",
			tokens: []Token{
				{Token: token.IDENT, Lit: `a`},
				{Token: token.ASSIGN, Lit: `=`},
				{Token: token.INT, Lit: `1`},
				{Token: token.IDENT, Lit: `b`},
				{Token: token.ASSIGN, Lit: `=`},
				{Token: token.INT, Lit: `2`},
				{Token: token.IDENT, Lit: `c`},
				{Token: token.ASSIGN, Lit: `=`},
				{Token: token.IDENT, Lit: `a`},
				{Token: token.ADD, Lit: `+`},
				{Token: token.IDENT, Lit: `b`},
				{Token: token.IDENT, Lit: `d`},
				{Token: token.ASSIGN, Lit: `=`},
				{Token: token.IDENT, Lit: `a`},
			},
			want: &ast.Program{
				Body: []ast.Statement{
					&ast.VariableDeclaration{
						Declarations: []*ast.VariableDeclarator{{
							ID: &ast.Identifier{
								Name: "a",
							},
							Init: &ast.IntegerLiteral{Value: 1},
						}},
					},
					&ast.VariableDeclaration{
						Declarations: []*ast.VariableDeclarator{{
							ID: &ast.Identifier{
								Name: "b",
							},
							Init: &ast.IntegerLiteral{Value: 2},
						}},
					},
					&ast.VariableDeclaration{
						Declarations: []*ast.VariableDeclarator{{
							ID: &ast.Identifier{
								Name: "c",
							},
							Init: &ast.BinaryExpression{
								Operator: ast.AdditionOperator,
								Left:     &ast.Identifier{Name: "a"},
								Right:    &ast.Identifier{Name: "b"},
							},
						}},
					},
					&ast.VariableDeclaration{
						Declarations: []*ast.VariableDeclarator{{
							ID: &ast.Identifier{
								Name: "d",
							},
							Init: &ast.Identifier{Name: "a"},
						}},
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fatalf := t.Fatalf
			if tt.skip {
				fatalf = t.Skipf
			}

			scanner := &Scanner{Tokens: tt.tokens}
			result, err := parser.NewAST(scanner)
			if err != nil {
				fatalf("unexpected error: %s", err)
			}

			if got, want := result, tt.want; !cmp.Equal(want, got, CompareOptions...) {
				fatalf("unexpected statement -want/+got\n%s", cmp.Diff(want, got, CompareOptions...))
			}
		})
	}
}

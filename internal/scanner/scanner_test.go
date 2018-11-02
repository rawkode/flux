package scanner_test

import (
	"testing"

	"github.com/influxdata/flux/internal/scanner"
	"github.com/influxdata/flux/internal/token"
)

type TokenPattern struct {
	s   string
	tok token.Token
	lit string
}

// common lists the patterns common for both scanning functions.
var common = []TokenPattern{
	{s: "0", tok: token.INT, lit: "0"},
	{s: "42", tok: token.INT, lit: "42"},
	{s: "317316873", tok: token.INT, lit: "317316873"},
	{s: "0.", tok: token.FLOAT, lit: "0."},
	{s: "72.40", tok: token.FLOAT, lit: "72.40"},
	{s: "072.40", tok: token.FLOAT, lit: "072.40"},
	{s: "2.71828", tok: token.FLOAT, lit: "2.71828"},
	{s: ".26", tok: token.FLOAT, lit: ".26"},
	{s: "1s", tok: token.DURATION, lit: "1s"},
	{s: "10d", tok: token.DURATION, lit: "10d"},
	{s: "1h15m", tok: token.DURATION, lit: "1h15m"},
	{s: "5w", tok: token.DURATION, lit: "5w"},
	{s: "1mo5d", tok: token.DURATION, lit: "1mo5d"},
	{s: "1952-01-25T12:35:51Z", tok: token.TIME, lit: "1952-01-25T12:35:51Z"},
	{s: "2018-08-15T13:36:23-07:00", tok: token.TIME, lit: "2018-08-15T13:36:23-07:00"},
	{s: "2009-10-15T09:00:00", tok: token.TIME, lit: "2009-10-15T09:00:00"},
	{s: "2018-01-01", tok: token.TIME, lit: "2018-01-01"},
	{s: `"abc"`, tok: token.STRING, lit: `"abc"`},
	{s: `"string with double \" quote"`, tok: token.STRING, lit: `"string with double \" quote"`},
	{s: `"string with backslash \\"`, tok: token.STRING, lit: `"string with backslash \\"`},
	{s: `"日本語"`, tok: token.STRING, lit: `"日本語"`},
	{s: `"\xe6\x97\xa5\xe6\x9c\xac\xe8\xaa\x9e"`, tok: token.STRING, lit: `"\xe6\x97\xa5\xe6\x9c\xac\xe8\xaa\x9e"`},
	{s: `a`, tok: token.IDENT, lit: `a`},
	{s: `_x`, tok: token.IDENT, lit: `_x`},
	{s: `longIdentifierName`, tok: token.IDENT, lit: `longIdentifierName`},
	{s: `αβ`, tok: token.IDENT, lit: `αβ`},
	{s: `and`, tok: token.AND, lit: `and`},
	{s: `or`, tok: token.OR, lit: `or`},
	{s: `not`, tok: token.NOT, lit: `not`},
	{s: `empty`, tok: token.EMPTY, lit: `empty`},
	{s: `in`, tok: token.IN, lit: `in`},
	{s: `import`, tok: token.IMPORT, lit: `import`},
	{s: `package`, tok: token.PACKAGE, lit: `package`},
	{s: `return`, tok: token.RETURN, lit: `return`},
	{s: `+`, tok: token.ADD, lit: `+`},
	{s: `-`, tok: token.SUB, lit: `-`},
	{s: `*`, tok: token.MUL, lit: `*`},
	// We skip div because the general parser can't tell the difference
	// between div and regex.
	{s: `%`, tok: token.MOD, lit: `%`},
	{s: `==`, tok: token.EQ, lit: `==`},
	{s: `<`, tok: token.LT, lit: `<`},
	{s: `>`, tok: token.GT, lit: `>`},
	{s: `<=`, tok: token.LTE, lit: `<=`},
	{s: `>=`, tok: token.GTE, lit: `>=`},
	{s: `!=`, tok: token.NEQ, lit: `!=`},
	{s: `=~`, tok: token.REGEXEQ, lit: `=~`},
	{s: `!~`, tok: token.REGEXNEQ, lit: `!~`},
	{s: `=`, tok: token.ASSIGN, lit: `=`},
	{s: `<-`, tok: token.ARROW, lit: `<-`},
	{s: `(`, tok: token.LPAREN, lit: `(`},
	{s: `)`, tok: token.RPAREN, lit: `)`},
	{s: `[`, tok: token.LBRACK, lit: `[`},
	{s: `]`, tok: token.RBRACK, lit: `]`},
	{s: `{`, tok: token.LBRACE, lit: `{`},
	{s: `}`, tok: token.RBRACE, lit: `}`},
	{s: `,`, tok: token.COMMA, lit: `,`},
	{s: `.`, tok: token.DOT, lit: `.`},
	{s: `:`, tok: token.COLON, lit: `:`},
	{s: `|>`, tok: token.PIPE, lit: `|>`},
}

// regex contains the regex patterns for the normal scan method.
var regex = []TokenPattern{
	{s: `/.*/`, tok: token.REGEX, lit: `/.*/`},
	{s: `/http:\/\/localhost:9999/`, tok: token.REGEX, lit: `/http:\/\/localhost:9999/`},
	{s: `/^\xe6\x97\xa5\xe6\x9c\xac\xe8\xaa\x9e(ZZ)?$/`, tok: token.REGEX, lit: `/^\xe6\x97\xa5\xe6\x9c\xac\xe8\xaa\x9e(ZZ)?$/`},
	{s: `/^日本語(ZZ)?$/`, tok: token.REGEX, lit: `/^日本語(ZZ)?$/`},
	{s: `/\\xZZ/`, tok: token.REGEX, lit: `/\\xZZ/`},
}

// noRegex contains the patterns to test when excluding regexes.
var noRegex = []TokenPattern{
	{s: `/`, tok: token.DIV, lit: `/`},
}

func patterns(patterns ...[]TokenPattern) []TokenPattern {
	sz := 0
	for _, a := range patterns {
		sz += len(a)
	}

	combined := make([]TokenPattern, 0, sz)
	for _, a := range patterns {
		combined = append(combined, a...)
	}
	return combined
}

func TestScanner_Scan(t *testing.T) {
	for _, tt := range patterns(common, regex) {
		t.Run(tt.s, func(t *testing.T) {
			s := scanner.New([]byte(tt.s))
			_, tok, lit := s.Scan()
			if want, got := tt.tok, tok; want != got {
				t.Errorf("unexpected token -want/+got\n\t- %d\n\t+ %d", want, got)
			}
			if want, got := tt.lit, lit; want != got {
				t.Errorf("unexpected literal -want/+got\n\t- %s\n\t+ %s", want, got)
			}

			// Expect an EOF token.
			if _, tok, _ := s.Scan(); tok != token.EOF {
				t.Errorf("expected eof token, got %d", tok)
			}
		})
	}
}

func TestScanner_ScanNoRegex(t *testing.T) {
	for _, tt := range patterns(common, noRegex) {
		t.Run(tt.s, func(t *testing.T) {
			s := scanner.New([]byte(tt.s))
			_, tok, lit := s.ScanNoRegex()
			if want, got := tt.tok, tok; want != got {
				t.Errorf("unexpected token -want/+got\n\t- %d\n\t+ %d", want, got)
			}
			if want, got := tt.lit, lit; want != got {
				t.Errorf("unexpected literal -want/+got\n\t- %s\n\t+ %s", want, got)
			}

			// Expect an EOF token.
			if _, tok, _ := s.ScanNoRegex(); tok != token.EOF {
				t.Errorf("expected eof token, got %d", tok)
			}
		})
	}
}

func TestScanner_Unread(t *testing.T) {
	s := scanner.New([]byte(`a /hello/`))
	_, tok, _ := s.Scan()
	if want, got := token.IDENT, tok; want != got {
		t.Fatalf("unexpected first token: %d", tok)
	}

	// First unread should read the same ident again.
	s.Unread()

	_, tok, _ = s.Scan()
	if want, got := token.IDENT, tok; want != got {
		t.Fatalf("unexpected token after first unread: %d", tok)
	}

	// Read the next token using the standard scan.
	_, tok, _ = s.Scan()
	if want, got := token.REGEX, tok; want != got {
		t.Fatalf("unexpected token after first unread: %d", tok)
	}

	// Unread should move back to the beginning and scanning without
	// regex should give us the division operator.
	s.Unread()
	_, tok, _ = s.ScanNoRegex()
	if want, got := token.DIV, tok; want != got {
		t.Fatalf("unexpected token after first unread: %d", tok)
	}

	// Unread twice and scan again should give us the regex again.
	s.Unread()
	s.Unread()
	_, tok, _ = s.Scan()
	if want, got := token.REGEX, tok; want != got {
		t.Fatalf("unexpected token after first unread: %d", tok)
	}
}

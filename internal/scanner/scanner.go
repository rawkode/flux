package scanner

import (
	"github.com/influxdata/flux/internal/token"
)

//go:generate ruby unicode2ragel.rb -e utf8 -o unicode.rl
//go:generate ragel -I. -Z scanner.rl -o scanner.gen.go
//go:generate sh -c "go fmt scanner.gen.go > /dev/null"

type Scanner struct {
	p, pe, eof  int
	ts, te, act int
	curline     int
	token       token.Token
	data        []byte
	reset       int
}

func New(data []byte) *Scanner {
	s := &Scanner{}
	s.Init(data)
	return s
}

func (s *Scanner) Init(data []byte) {
	s.p, s.pe, s.eof = 0, len(data), len(data)
	s.data = data
	s.curline = 1
	s.init()
}

func (s *Scanner) Scan() (pos token.Pos, tok token.Token, lit string) {
	return s.scan(flux_en_main)
}

func (s *Scanner) ScanNoRegex() (pos token.Pos, tok token.Token, lit string) {
	return s.scan(flux_en_main_no_regex)
}

// Unread will reset the Scanner to go back to the Scanner's location
// before the last Scan or ScanNoRegex call.
func (s *Scanner) Unread() {
	s.p = s.reset
}

func (s *Scanner) scan(cs int) (pos token.Pos, tok token.Token, lit string) {
	s.reset = s.p
	s.token = token.ILLEGAL
	if es := s.exec(cs); es == flux_error {
		return 0, token.ILLEGAL, ""
	} else if s.token == token.ILLEGAL && s.p == s.eof {
		return 0, token.EOF, ""
	}
	return 0, s.token, string(s.data[s.ts:s.te])
}

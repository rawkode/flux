package parser

import (
	"github.com/influxdata/flux/ast"
)

type Program struct {
	Root      ast.Program
	Statement ParseNode
}

func (p Program) Parse(s Scanner) (ParseNode, bool) {
	if p.Statement == nil {
		p.Statement = Statement{}
	}

	if next, ok := p.Statement.Parse(s); ok {
		p.Statement = next
		return p, true
	}

	// Materialize the statement.
	stmt, err := p.Statement.Get()
	if err != nil {
		// The statement did not error when reading from the
		// scanner, but it is also not ready. If the statement
		// is in a terminal state, then return the error.
		if IsTerminal(p.Statement) {
			return Error(err), true
		}

		// This is a non-terminal error so say we could not
		// continue and let the calling code figure it out.
		return nil, false
	}

	// Now create a new statement and attempt to use it.
	// If we can't, then maybe we got an EOF on the last one
	// and we shouldn't have continued.
	p.Statement = Statement{}
	if next, ok := p.Statement.Parse(s); ok {
		p.Root.Body = append(p.Root.Body, stmt.(ast.Statement))
		p.Statement = next
		return p, true
	}
	return nil, false
}

func (p Program) Get() (ast.Node, error) {
	if p.Statement != nil {
		stmt, err := p.Statement.Get()
		if err != nil {
			return nil, err
		}
		p.Root.Body = append(p.Root.Body, stmt.(ast.Statement))
	}
	return &p.Root, nil
}

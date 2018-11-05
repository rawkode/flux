package parser

import (
	"fmt"

	"github.com/influxdata/flux/ast"
)

// errorNode is returned when a terminal error is encountered.
type errorNode struct {
	Err error
}

func (e errorNode) Parse(s Scanner) (ParseNode, bool) {
	return nil, false
}

func (e errorNode) Get() (ast.Node, error) {
	return nil, e.Err
}

func (e errorNode) IsTerminal() bool {
	return true
}

func Error(err error) ParseNode {
	return errorNode{Err: err}
}

func Errorf(msg string, v ...interface{}) ParseNode {
	return Error(fmt.Errorf(msg, v...))
}

// Package parser implements a parser for flux.
//
// The parser is composed of a series of ParseNode objects. The ParseNode has a simple interface.
// It accepts the Scanner which is defined within this package so that alternative scanning
// implementations can be swapped out and different ones can be experimented with.
//
// Each ParseNode takes the Scanner as input. If a ParseNode is able to do something with the next
// token, it will consume the token and produce a new ParseNode that will move to the next stage
// of parsing. For efficiency, a ParseNode may consume more than one token.
//
// Once a ParseNode is not able to consume any more input from the Scanner, it indicates that the
// parsing is in one of two states: terminal or non-terminal. A terminal state means that no more
// input will be accepted no matter what gets fed into the parser. This is most commonly used with
// error states, but it may also indicate that a subtree in the parse tree has finished. A
// non-terminal state means that the system could continue feeding input to the node.
//
// After reaching either a terminal or non-terminal state, the code calling the parser can invoke
// the Get method on the ParseNode. The Get method will construct the ast.Node if it is able to
// and it will return an error if the parser ended in a terminal error state.
//
// The state within each ParseNode is immutable after it has been returned. That means that each stage
// of the parsing could be retained and the state of the Scanner could be preserved for each one
// to be able to replay the entire process. While the state is immutable, the ParseNode's are not
// thread or memory-safe. If you use an older ParseNode, it may use the same memory as a different
// ParseNode for things involving slices and maps. The returned nodes are shallow copies.
//
// TODO(jsternberg): I need to find a way to report the difference between a non-final state error
// and an error-error. Most likely through a custom error type that will both signal that the error
// is recoverable and indicate ways to correct it (such as through completion suggestions maybe?).
package parser

import (
	"github.com/influxdata/flux/ast"
	"github.com/influxdata/flux/internal/token"
)

// Scanner defines the interface for reading a stream of tokens.
type Scanner interface {
	// Scan will scan the next token.
	Scan() (token.Pos, token.Token, string)

	// ScanNoRegex will scan the next token, but exclude any
	// regex literals.
	ScanNoRegex() (token.Pos, token.Token, string)

	// Unread will unread back to the previous location within the Scanner.
	// This can only be called once so the maximum lookahead is one.
	Unread()
}

// ParseNode refers to a portion of the parse tree. It holds the current location
// within the parser so that the parser can be interrupted and continued
// as the controlling program sees is needed. It may also contain information about
// what tokens are expected and which tokens have been encountered to aid with
// syntax highlighting and code completion.
type ParseNode interface {
	// Parse will use the Scanner to parse the tokens into another ParseNode.
	// If this ParseNode is able to successfully consume at least one token from the Scanner,
	// then the second return value will be true and a new ParseNode will be returned.
	// If the ParseNode does not consume at least one token from the Scanner, then it
	// will not return a next node and it will return false. This may happen because the
	// ParseNode encountered an EOF, it might be complete and there are no additional tokens
	// to consume, or the next token may be invalid but the current node is valid without it.
	Parse(s Scanner) (next ParseNode, ok bool)

	// Get will retrieve the AST node that has been constructed from the ParseNode.
	Get() (ast.Node, error)
}

// TerminalNode is a (potentially) terminal ParseNode.
type TerminalNode interface {
	ParseNode

	// IsTerminal indicates if this is a terminal node.
	IsTerminal() bool
}

// IsTerminal will determine if a ParseNode is terminal. If the ParseNode does not
// implement the TerminalNode interface, it is assumed to be a non-terminal node.
func IsTerminal(n ParseNode) bool {
	if n, ok := n.(TerminalNode); ok {
		return n.IsTerminal()
	}
	return false
}

// Feed will continuous feed the Scanner to the ParseNode until Parse returns false.
// It will then return the last ParseNode in the series.
func Feed(n ParseNode, s Scanner) ParseNode {
	for {
		next, ok := n.Parse(s)
		if !ok {
			return n
		}
		n = next
	}
}

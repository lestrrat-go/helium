package xpath1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLexerBasicPath(t *testing.T) {
	l, err := newLexer("/foo/bar")
	require.NoError(t, err)

	tokens := l.Tokens()
	require.Len(t, tokens, 4)
	require.Equal(t, TokenSlash, tokens[0].Type)
	require.Equal(t, TokenName, tokens[1].Type)
	require.Equal(t, "foo", tokens[1].Value)
	require.Equal(t, TokenSlash, tokens[2].Type)
	require.Equal(t, TokenName, tokens[3].Type)
	require.Equal(t, "bar", tokens[3].Value)
}

func TestLexerDoubleSlash(t *testing.T) {
	l, err := newLexer("//div")
	require.NoError(t, err)

	tokens := l.Tokens()
	require.Len(t, tokens, 2)
	require.Equal(t, TokenSlashSlash, tokens[0].Type)
	require.Equal(t, TokenName, tokens[1].Type)
	require.Equal(t, "div", tokens[1].Value)
}

func TestLexerAttribute(t *testing.T) {
	l, err := newLexer("@id")
	require.NoError(t, err)

	tokens := l.Tokens()
	require.Len(t, tokens, 2)
	require.Equal(t, TokenAt, tokens[0].Type)
	require.Equal(t, TokenName, tokens[1].Type)
	require.Equal(t, "id", tokens[1].Value)
}

func TestLexerPredicate(t *testing.T) {
	l, err := newLexer("item[3]")
	require.NoError(t, err)

	tokens := l.Tokens()
	require.Len(t, tokens, 4)
	require.Equal(t, TokenName, tokens[0].Type)
	require.Equal(t, "item", tokens[0].Value)
	require.Equal(t, TokenLBracket, tokens[1].Type)
	require.Equal(t, TokenNumber, tokens[2].Type)
	require.Equal(t, "3", tokens[2].Value)
	require.Equal(t, TokenRBracket, tokens[3].Type)
}

func TestLexerStringLiterals(t *testing.T) {
	l, err := newLexer(`"hello"`)
	require.NoError(t, err)
	tokens := l.Tokens()
	require.Len(t, tokens, 1)
	require.Equal(t, TokenString, tokens[0].Type)
	require.Equal(t, "hello", tokens[0].Value)

	l, err = newLexer(`'world'`)
	require.NoError(t, err)
	tokens = l.Tokens()
	require.Len(t, tokens, 1)
	require.Equal(t, TokenString, tokens[0].Type)
	require.Equal(t, "world", tokens[0].Value)
}

func TestLexerNumber(t *testing.T) {
	l, err := newLexer("42")
	require.NoError(t, err)
	tokens := l.Tokens()
	require.Len(t, tokens, 1)
	require.Equal(t, TokenNumber, tokens[0].Type)
	require.Equal(t, "42", tokens[0].Value)

	l, err = newLexer("3.14")
	require.NoError(t, err)
	tokens = l.Tokens()
	require.Len(t, tokens, 1)
	require.Equal(t, TokenNumber, tokens[0].Type)
	require.Equal(t, "3.14", tokens[0].Value)
}

func TestLexerOperators(t *testing.T) {
	// "a and b" — "and" follows a name, so it's an operator
	l, err := newLexer("a and b")
	require.NoError(t, err)
	tokens := l.Tokens()
	require.Len(t, tokens, 3)
	require.Equal(t, TokenName, tokens[0].Type)
	require.Equal(t, TokenAnd, tokens[1].Type)
	require.Equal(t, TokenName, tokens[2].Type)
}

func TestLexerOperatorKeywords(t *testing.T) {
	l, err := newLexer("a or b")
	require.NoError(t, err)
	tokens := l.Tokens()
	require.Len(t, tokens, 3)
	require.Equal(t, TokenOr, tokens[1].Type)

	l, err = newLexer("a mod b")
	require.NoError(t, err)
	tokens = l.Tokens()
	require.Equal(t, TokenMod, tokens[1].Type)

	l, err = newLexer("a div b")
	require.NoError(t, err)
	tokens = l.Tokens()
	require.Equal(t, TokenDiv, tokens[1].Type)
}

func TestLexerAxisNotation(t *testing.T) {
	l, err := newLexer("child::para")
	require.NoError(t, err)
	tokens := l.Tokens()
	require.Len(t, tokens, 3)
	require.Equal(t, TokenName, tokens[0].Type)
	require.Equal(t, "child", tokens[0].Value)
	require.Equal(t, TokenColonColon, tokens[1].Type)
	require.Equal(t, TokenName, tokens[2].Type)
	require.Equal(t, "para", tokens[2].Value)
}

func TestLexerDots(t *testing.T) {
	l, err := newLexer("../child")
	require.NoError(t, err)
	tokens := l.Tokens()
	require.Len(t, tokens, 3)
	require.Equal(t, TokenDotDot, tokens[0].Type)
	require.Equal(t, TokenSlash, tokens[1].Type)
	require.Equal(t, TokenName, tokens[2].Type)
}

func TestLexerVariable(t *testing.T) {
	l, err := newLexer("$x")
	require.NoError(t, err)
	tokens := l.Tokens()
	require.Len(t, tokens, 1)
	require.Equal(t, TokenVariableRef, tokens[0].Type)
	require.Equal(t, "x", tokens[0].Value)
}

func TestLexerComparison(t *testing.T) {
	l, err := newLexer("a != b")
	require.NoError(t, err)
	tokens := l.Tokens()
	require.Len(t, tokens, 3)
	require.Equal(t, TokenNotEquals, tokens[1].Type)

	l, err = newLexer("a <= b")
	require.NoError(t, err)
	tokens = l.Tokens()
	require.Equal(t, TokenLessEq, tokens[1].Type)

	l, err = newLexer("a >= b")
	require.NoError(t, err)
	tokens = l.Tokens()
	require.Equal(t, TokenGreaterEq, tokens[1].Type)
}

func TestLexerFunctionCall(t *testing.T) {
	l, err := newLexer("count(//item)")
	require.NoError(t, err)
	tokens := l.Tokens()
	require.Len(t, tokens, 5)
	require.Equal(t, TokenName, tokens[0].Type)
	require.Equal(t, "count", tokens[0].Value)
	require.Equal(t, TokenLParen, tokens[1].Type)
	require.Equal(t, TokenSlashSlash, tokens[2].Type)
	require.Equal(t, TokenName, tokens[3].Type)
	require.Equal(t, TokenRParen, tokens[4].Type)
}

func TestLexerQName(t *testing.T) {
	l, err := newLexer("ns:elem")
	require.NoError(t, err)
	tokens := l.Tokens()
	require.Len(t, tokens, 3)
	require.Equal(t, TokenName, tokens[0].Type)
	require.Equal(t, "ns", tokens[0].Value)
	require.Equal(t, TokenColon, tokens[1].Type)
	require.Equal(t, TokenName, tokens[2].Type)
	require.Equal(t, "elem", tokens[2].Value)
}

func TestLexerNamespaceWildcard(t *testing.T) {
	l, err := newLexer("ns:*")
	require.NoError(t, err)
	tokens := l.Tokens()
	require.Len(t, tokens, 3)
	require.Equal(t, TokenName, tokens[0].Type)
	require.Equal(t, "ns", tokens[0].Value)
	require.Equal(t, TokenColon, tokens[1].Type)
	require.Equal(t, TokenStar, tokens[2].Type)
}

func TestLexerComplex(t *testing.T) {
	l, err := newLexer(`/bookstore/book[price>35.00]/title`)
	require.NoError(t, err)
	tokens := l.Tokens()

	expected := []struct {
		typ TokenType
		val string
	}{
		{TokenSlash, "/"},
		{TokenName, "bookstore"},
		{TokenSlash, "/"},
		{TokenName, "book"},
		{TokenLBracket, "["},
		{TokenName, "price"},
		{TokenGreater, ">"},
		{TokenNumber, "35.00"},
		{TokenRBracket, "]"},
		{TokenSlash, "/"},
		{TokenName, "title"},
	}

	require.Len(t, tokens, len(expected))
	for i, e := range expected {
		require.Equal(t, e.typ, tokens[i].Type, "token %d type", i)
		require.Equal(t, e.val, tokens[i].Value, "token %d value", i)
	}
}

func TestLexerUnion(t *testing.T) {
	l, err := newLexer("a | b")
	require.NoError(t, err)
	tokens := l.Tokens()
	require.Len(t, tokens, 3)
	require.Equal(t, TokenPipe, tokens[1].Type)
}

func TestLexerAndAsName(t *testing.T) {
	// "and" at the start of an expression is a name, not an operator
	l, err := newLexer("and")
	require.NoError(t, err)
	tokens := l.Tokens()
	require.Len(t, tokens, 1)
	require.Equal(t, TokenName, tokens[0].Type)
	require.Equal(t, "and", tokens[0].Value)
}

func TestLexerStar(t *testing.T) {
	l, err := newLexer("child::*")
	require.NoError(t, err)
	tokens := l.Tokens()
	require.Len(t, tokens, 3)
	require.Equal(t, TokenStar, tokens[2].Type)
}

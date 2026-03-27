package lexer_test

import (
	"testing"

	"github.com/lestrrat-go/helium/internal/xpath1/lexer"
	"github.com/stretchr/testify/require"
)

func TestLexerBasicPath(t *testing.T) {
	l, err := lexer.New("/foo/bar")
	require.NoError(t, err)

	tokens := l.Tokens()
	require.Len(t, tokens, 4)
	require.Equal(t, lexer.TokenSlash, tokens[0].Type)
	require.Equal(t, lexer.TokenName, tokens[1].Type)
	require.Equal(t, "foo", tokens[1].Value)
	require.Equal(t, lexer.TokenSlash, tokens[2].Type)
	require.Equal(t, lexer.TokenName, tokens[3].Type)
	require.Equal(t, "bar", tokens[3].Value)
}

func TestLexerDoubleSlash(t *testing.T) {
	l, err := lexer.New("//div")
	require.NoError(t, err)

	tokens := l.Tokens()
	require.Len(t, tokens, 2)
	require.Equal(t, lexer.TokenSlashSlash, tokens[0].Type)
	require.Equal(t, lexer.TokenName, tokens[1].Type)
	require.Equal(t, "div", tokens[1].Value)
}

func TestLexerAttribute(t *testing.T) {
	l, err := lexer.New("@id")
	require.NoError(t, err)

	tokens := l.Tokens()
	require.Len(t, tokens, 2)
	require.Equal(t, lexer.TokenAt, tokens[0].Type)
	require.Equal(t, lexer.TokenName, tokens[1].Type)
	require.Equal(t, "id", tokens[1].Value)
}

func TestLexerPredicate(t *testing.T) {
	l, err := lexer.New("item[3]")
	require.NoError(t, err)

	tokens := l.Tokens()
	require.Len(t, tokens, 4)
	require.Equal(t, lexer.TokenName, tokens[0].Type)
	require.Equal(t, "item", tokens[0].Value)
	require.Equal(t, lexer.TokenLBracket, tokens[1].Type)
	require.Equal(t, lexer.TokenNumber, tokens[2].Type)
	require.Equal(t, "3", tokens[2].Value)
	require.Equal(t, lexer.TokenRBracket, tokens[3].Type)
}

func TestLexerStringLiterals(t *testing.T) {
	l, err := lexer.New(`"hello"`)
	require.NoError(t, err)
	tokens := l.Tokens()
	require.Len(t, tokens, 1)
	require.Equal(t, lexer.TokenString, tokens[0].Type)
	require.Equal(t, "hello", tokens[0].Value)

	l, err = lexer.New(`'world'`)
	require.NoError(t, err)
	tokens = l.Tokens()
	require.Len(t, tokens, 1)
	require.Equal(t, lexer.TokenString, tokens[0].Type)
	require.Equal(t, "world", tokens[0].Value)
}

func TestLexerNumber(t *testing.T) {
	l, err := lexer.New("42")
	require.NoError(t, err)
	tokens := l.Tokens()
	require.Len(t, tokens, 1)
	require.Equal(t, lexer.TokenNumber, tokens[0].Type)
	require.Equal(t, "42", tokens[0].Value)

	l, err = lexer.New("3.14")
	require.NoError(t, err)
	tokens = l.Tokens()
	require.Len(t, tokens, 1)
	require.Equal(t, lexer.TokenNumber, tokens[0].Type)
	require.Equal(t, "3.14", tokens[0].Value)
}

func TestLexerOperators(t *testing.T) {
	// "a and b" — "and" follows a name, so it's an operator
	l, err := lexer.New("a and b")
	require.NoError(t, err)
	tokens := l.Tokens()
	require.Len(t, tokens, 3)
	require.Equal(t, lexer.TokenName, tokens[0].Type)
	require.Equal(t, lexer.TokenAnd, tokens[1].Type)
	require.Equal(t, lexer.TokenName, tokens[2].Type)
}

func TestLexerOperatorKeywords(t *testing.T) {
	l, err := lexer.New("a or b")
	require.NoError(t, err)
	tokens := l.Tokens()
	require.Len(t, tokens, 3)
	require.Equal(t, lexer.TokenOr, tokens[1].Type)

	l, err = lexer.New("a mod b")
	require.NoError(t, err)
	tokens = l.Tokens()
	require.Equal(t, lexer.TokenMod, tokens[1].Type)

	l, err = lexer.New("a div b")
	require.NoError(t, err)
	tokens = l.Tokens()
	require.Equal(t, lexer.TokenDiv, tokens[1].Type)
}

func TestLexerAxisNotation(t *testing.T) {
	l, err := lexer.New("child::para")
	require.NoError(t, err)
	tokens := l.Tokens()
	require.Len(t, tokens, 3)
	require.Equal(t, lexer.TokenName, tokens[0].Type)
	require.Equal(t, "child", tokens[0].Value)
	require.Equal(t, lexer.TokenColonColon, tokens[1].Type)
	require.Equal(t, lexer.TokenName, tokens[2].Type)
	require.Equal(t, "para", tokens[2].Value)
}

func TestLexerDots(t *testing.T) {
	l, err := lexer.New("../child")
	require.NoError(t, err)
	tokens := l.Tokens()
	require.Len(t, tokens, 3)
	require.Equal(t, lexer.TokenDotDot, tokens[0].Type)
	require.Equal(t, lexer.TokenSlash, tokens[1].Type)
	require.Equal(t, lexer.TokenName, tokens[2].Type)
}

func TestLexerVariable(t *testing.T) {
	l, err := lexer.New("$x")
	require.NoError(t, err)
	tokens := l.Tokens()
	require.Len(t, tokens, 1)
	require.Equal(t, lexer.TokenVariableRef, tokens[0].Type)
	require.Equal(t, "x", tokens[0].Value)
}

func TestLexerComparison(t *testing.T) {
	l, err := lexer.New("a != b")
	require.NoError(t, err)
	tokens := l.Tokens()
	require.Len(t, tokens, 3)
	require.Equal(t, lexer.TokenNotEquals, tokens[1].Type)

	l, err = lexer.New("a <= b")
	require.NoError(t, err)
	tokens = l.Tokens()
	require.Equal(t, lexer.TokenLessEq, tokens[1].Type)

	l, err = lexer.New("a >= b")
	require.NoError(t, err)
	tokens = l.Tokens()
	require.Equal(t, lexer.TokenGreaterEq, tokens[1].Type)
}

func TestLexerFunctionCall(t *testing.T) {
	l, err := lexer.New("count(//item)")
	require.NoError(t, err)
	tokens := l.Tokens()
	require.Len(t, tokens, 5)
	require.Equal(t, lexer.TokenName, tokens[0].Type)
	require.Equal(t, "count", tokens[0].Value)
	require.Equal(t, lexer.TokenLParen, tokens[1].Type)
	require.Equal(t, lexer.TokenSlashSlash, tokens[2].Type)
	require.Equal(t, lexer.TokenName, tokens[3].Type)
	require.Equal(t, lexer.TokenRParen, tokens[4].Type)
}

func TestLexerQName(t *testing.T) {
	l, err := lexer.New("ns:elem")
	require.NoError(t, err)
	tokens := l.Tokens()
	require.Len(t, tokens, 3)
	require.Equal(t, lexer.TokenName, tokens[0].Type)
	require.Equal(t, "ns", tokens[0].Value)
	require.Equal(t, lexer.TokenColon, tokens[1].Type)
	require.Equal(t, lexer.TokenName, tokens[2].Type)
	require.Equal(t, "elem", tokens[2].Value)
}

func TestLexerNamespaceWildcard(t *testing.T) {
	l, err := lexer.New("ns:*")
	require.NoError(t, err)
	tokens := l.Tokens()
	require.Len(t, tokens, 3)
	require.Equal(t, lexer.TokenName, tokens[0].Type)
	require.Equal(t, "ns", tokens[0].Value)
	require.Equal(t, lexer.TokenColon, tokens[1].Type)
	require.Equal(t, lexer.TokenStar, tokens[2].Type)
}

func TestLexerComplex(t *testing.T) {
	l, err := lexer.New(`/bookstore/book[price>35.00]/title`)
	require.NoError(t, err)
	tokens := l.Tokens()

	expected := []struct {
		typ lexer.TokenType
		val string
	}{
		{lexer.TokenSlash, "/"},
		{lexer.TokenName, "bookstore"},
		{lexer.TokenSlash, "/"},
		{lexer.TokenName, "book"},
		{lexer.TokenLBracket, "["},
		{lexer.TokenName, "price"},
		{lexer.TokenGreater, ">"},
		{lexer.TokenNumber, "35.00"},
		{lexer.TokenRBracket, "]"},
		{lexer.TokenSlash, "/"},
		{lexer.TokenName, "title"},
	}

	require.Len(t, tokens, len(expected))
	for i, e := range expected {
		require.Equal(t, e.typ, tokens[i].Type, "token %d type", i)
		require.Equal(t, e.val, tokens[i].Value, "token %d value", i)
	}
}

func TestLexerUnion(t *testing.T) {
	l, err := lexer.New("a | b")
	require.NoError(t, err)
	tokens := l.Tokens()
	require.Len(t, tokens, 3)
	require.Equal(t, lexer.TokenPipe, tokens[1].Type)
}

func TestLexerAndAsName(t *testing.T) {
	// "and" at the start of an expression is a name, not an operator
	l, err := lexer.New("and")
	require.NoError(t, err)
	tokens := l.Tokens()
	require.Len(t, tokens, 1)
	require.Equal(t, lexer.TokenName, tokens[0].Type)
	require.Equal(t, "and", tokens[0].Value)
}

func TestLexerStar(t *testing.T) {
	l, err := lexer.New("child::*")
	require.NoError(t, err)
	tokens := l.Tokens()
	require.Len(t, tokens, 3)
	require.Equal(t, lexer.TokenStar, tokens[2].Type)
}

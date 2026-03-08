package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

func tokenTypes(input string) ([]xpath3.TokenType, error) {
	l, err := xpath3.NewLexerForTesting(input)
	if err != nil {
		return nil, err
	}
	var types []xpath3.TokenType
	for _, t := range l.Tokens() {
		types = append(types, t.Type)
	}
	return types, nil
}

func TestLexerBasicTokens(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect []xpath3.TokenType
	}{
		{
			name:  "simple path",
			input: "/a/b",
			expect: []xpath3.TokenType{
				xpath3.TokenSlash, xpath3.TokenName, xpath3.TokenSlash, xpath3.TokenName,
			},
		},
		{
			name:  "descendant-or-self",
			input: "//a",
			expect: []xpath3.TokenType{
				xpath3.TokenSlashSlash, xpath3.TokenName,
			},
		},
		{
			name:  "predicate",
			input: "a[1]",
			expect: []xpath3.TokenType{
				xpath3.TokenName, xpath3.TokenLBracket, xpath3.TokenNumber, xpath3.TokenRBracket,
			},
		},
		{
			name:  "function call",
			input: "count(//x)",
			expect: []xpath3.TokenType{
				xpath3.TokenName, xpath3.TokenLParen, xpath3.TokenSlashSlash, xpath3.TokenName, xpath3.TokenRParen,
			},
		},
		{
			name:  "variable ref",
			input: "$x + $y",
			expect: []xpath3.TokenType{
				xpath3.TokenVariableRef, xpath3.TokenPlus, xpath3.TokenVariableRef,
			},
		},
		{
			name:  "string literal",
			input: `"hello"`,
			expect: []xpath3.TokenType{
				xpath3.TokenString,
			},
		},
		{
			name:  "axis specifier",
			input: "child::a",
			expect: []xpath3.TokenType{
				xpath3.TokenName, xpath3.TokenColonColon, xpath3.TokenName,
			},
		},
		{
			name:  "bare colon",
			input: `"key" : "value"`,
			expect: []xpath3.TokenType{
				xpath3.TokenString, xpath3.TokenColon, xpath3.TokenString,
			},
		},
		{
			name:  "qname",
			input: "ns:elem",
			expect: []xpath3.TokenType{
				xpath3.TokenName, xpath3.TokenColon, xpath3.TokenName,
			},
		},
		{
			name:  "wildcard prefix",
			input: "ns:*",
			expect: []xpath3.TokenType{
				xpath3.TokenName, xpath3.TokenColon, xpath3.TokenStar,
			},
		},
		{
			name:  "attribute shorthand",
			input: "@id",
			expect: []xpath3.TokenType{
				xpath3.TokenAt, xpath3.TokenName,
			},
		},
		{
			name:  "dot and dotdot",
			input: "../.",
			expect: []xpath3.TokenType{
				xpath3.TokenDotDot, xpath3.TokenSlash, xpath3.TokenDot,
			},
		},
		{
			name:  "number with decimal",
			input: "3.14",
			expect: []xpath3.TokenType{
				xpath3.TokenNumber,
			},
		},
		{
			name:  "scientific notation",
			input: "1.5E-3",
			expect: []xpath3.TokenType{
				xpath3.TokenNumber,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tokenTypes(tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.expect, got)
		})
	}
}

func TestLexerXPath3Tokens(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect []xpath3.TokenType
	}{
		{
			name:  "concat operator",
			input: `"a" || "b"`,
			expect: []xpath3.TokenType{
				xpath3.TokenString, xpath3.TokenConcat, xpath3.TokenString,
			},
		},
		{
			name:  "arrow operator",
			input: "$x => upper-case()",
			expect: []xpath3.TokenType{
				xpath3.TokenVariableRef, xpath3.TokenArrow, xpath3.TokenName,
				xpath3.TokenLParen, xpath3.TokenRParen,
			},
		},
		{
			name:  "bang operator",
			input: "//item ! name()",
			expect: []xpath3.TokenType{
				xpath3.TokenSlashSlash, xpath3.TokenName, xpath3.TokenBang,
				xpath3.TokenName, xpath3.TokenLParen, xpath3.TokenRParen,
			},
		},
		{
			name:  "hash for named function ref",
			input: "fn:upper-case#1",
			expect: []xpath3.TokenType{
				xpath3.TokenName, xpath3.TokenColon, xpath3.TokenName,
				xpath3.TokenHash, xpath3.TokenNumber,
			},
		},
		{
			name:  "question mark lookup",
			input: "$map?key",
			expect: []xpath3.TokenType{
				xpath3.TokenVariableRef, xpath3.TokenQMark, xpath3.TokenName,
			},
		},
		{
			name:  "braces",
			input: "map { }",
			expect: []xpath3.TokenType{
				xpath3.TokenMap, xpath3.TokenLBrace, xpath3.TokenRBrace,
			},
		},
		{
			name:  "not equals with bang",
			input: "1 != 2",
			expect: []xpath3.TokenType{
				xpath3.TokenNumber, xpath3.TokenNotEquals, xpath3.TokenNumber,
			},
		},
		{
			name:  "value comparison eq",
			input: "$x eq $y",
			expect: []xpath3.TokenType{
				xpath3.TokenVariableRef, xpath3.TokenEq, xpath3.TokenVariableRef,
			},
		},
		{
			name:  "value comparison ne",
			input: "1 ne 2",
			expect: []xpath3.TokenType{
				xpath3.TokenNumber, xpath3.TokenNe, xpath3.TokenNumber,
			},
		},
		{
			name:  "to range",
			input: "1 to 10",
			expect: []xpath3.TokenType{
				xpath3.TokenNumber, xpath3.TokenTo, xpath3.TokenNumber,
			},
		},
		{
			name:  "idiv operator",
			input: "10 idiv 3",
			expect: []xpath3.TokenType{
				xpath3.TokenNumber, xpath3.TokenIdiv, xpath3.TokenNumber,
			},
		},
		{
			name:  "intersect except",
			input: "$a intersect $b except $c",
			expect: []xpath3.TokenType{
				xpath3.TokenVariableRef, xpath3.TokenIntersect,
				xpath3.TokenVariableRef, xpath3.TokenExcept,
				xpath3.TokenVariableRef,
			},
		},
		{
			name:  "union keyword",
			input: "$a union $b",
			expect: []xpath3.TokenType{
				xpath3.TokenVariableRef, xpath3.TokenUnion, xpath3.TokenVariableRef,
			},
		},
		{
			name:  "for expression",
			input: "for $x in 1 return $x",
			expect: []xpath3.TokenType{
				xpath3.TokenFor, xpath3.TokenVariableRef, xpath3.TokenIn,
				xpath3.TokenNumber, xpath3.TokenReturn, xpath3.TokenVariableRef,
			},
		},
		{
			name:  "let expression",
			input: "let $x := 1 return $x",
			expect: []xpath3.TokenType{
				xpath3.TokenLet, xpath3.TokenVariableRef, xpath3.TokenColon,
				xpath3.TokenEquals, xpath3.TokenNumber, xpath3.TokenReturn,
				xpath3.TokenVariableRef,
			},
		},
		{
			name:  "if then else",
			input: "if (true()) then 1 else 2",
			expect: []xpath3.TokenType{
				xpath3.TokenIf, xpath3.TokenLParen, xpath3.TokenName,
				xpath3.TokenLParen, xpath3.TokenRParen, xpath3.TokenRParen,
				xpath3.TokenThen, xpath3.TokenNumber, xpath3.TokenElse, xpath3.TokenNumber,
			},
		},
		{
			name:  "some every satisfies",
			input: "some $x in //a satisfies $x > 0",
			expect: []xpath3.TokenType{
				xpath3.TokenSome, xpath3.TokenVariableRef, xpath3.TokenIn,
				xpath3.TokenSlashSlash, xpath3.TokenName,
				xpath3.TokenSatisfies, xpath3.TokenVariableRef,
				xpath3.TokenGreater, xpath3.TokenNumber,
			},
		},
		{
			name:  "instance of",
			input: "$x instance of xs:integer",
			expect: []xpath3.TokenType{
				xpath3.TokenVariableRef, xpath3.TokenInstanceOf, xpath3.TokenOf,
				xpath3.TokenName, xpath3.TokenColon, xpath3.TokenName,
			},
		},
		{
			name:  "cast as",
			input: "$x cast as xs:double",
			expect: []xpath3.TokenType{
				xpath3.TokenVariableRef, xpath3.TokenCastAs, xpath3.TokenAs,
				xpath3.TokenName, xpath3.TokenColon, xpath3.TokenName,
			},
		},
		{
			name:  "try catch",
			input: "try { 1 } catch * { 0 }",
			expect: []xpath3.TokenType{
				xpath3.TokenTry, xpath3.TokenLBrace, xpath3.TokenNumber, xpath3.TokenRBrace,
				xpath3.TokenCatch, xpath3.TokenStar, xpath3.TokenLBrace,
				xpath3.TokenNumber, xpath3.TokenRBrace,
			},
		},
		{
			name:  "inline function",
			input: "function($x) { $x + 1 }",
			expect: []xpath3.TokenType{
				xpath3.TokenFunction, xpath3.TokenLParen, xpath3.TokenVariableRef,
				xpath3.TokenRParen, xpath3.TokenLBrace,
				xpath3.TokenVariableRef, xpath3.TokenPlus, xpath3.TokenNumber,
				xpath3.TokenRBrace,
			},
		},
		{
			name:  "array square bracket",
			input: "[1, 2, 3]",
			expect: []xpath3.TokenType{
				xpath3.TokenLBracket, xpath3.TokenNumber, xpath3.TokenComma,
				xpath3.TokenNumber, xpath3.TokenComma, xpath3.TokenNumber,
				xpath3.TokenRBracket,
			},
		},
		{
			name:  "doubled quote escape",
			input: `"it''s"`,
			expect: []xpath3.TokenType{
				xpath3.TokenString,
			},
		},
		{
			name:  "keyword as name outside operator context",
			input: "div",
			expect: []xpath3.TokenType{
				xpath3.TokenName,
			},
		},
		{
			name:  "keyword in operator context",
			input: "1 div 2",
			expect: []xpath3.TokenType{
				xpath3.TokenNumber, xpath3.TokenDiv, xpath3.TokenNumber,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tokenTypes(tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.expect, got)
		})
	}
}

func TestLexerStringEscape(t *testing.T) {
	l, err := xpath3.NewLexerForTesting(`"it""s"`)
	require.NoError(t, err)
	tokens := l.Tokens()
	require.Len(t, tokens, 1)
	require.Equal(t, `it"s`, tokens[0].Value)
}

func TestLexerErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"unterminated string", `"hello`},
		{"invalid char", "~"},
		{"dollar without name", "$"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := xpath3.NewLexerForTesting(tc.input)
			require.Error(t, err)
		})
	}
}

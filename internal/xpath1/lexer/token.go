package lexer

import "fmt"

// TokenType identifies the kind of lexical token.
type TokenType int

// TokenEOF marks the end of input. TokenNumber, TokenString, TokenName,
// TokenStar, and TokenVariableRef represent literals and names.
// TokenSlash through TokenDiv represent operators.
// TokenLParen through TokenColon represent punctuation symbols.
const (
	TokenEOF         TokenType = iota // end of input
	TokenNumber                       // 42, 3.14
	TokenString                       // "hello", 'hello'
	TokenName                         // NCName (foo, bar)
	TokenStar                         // * (wildcard name test or multiply)
	TokenVariableRef                  // $name

	TokenSlash      // /
	TokenSlashSlash // //
	TokenPipe       // |
	TokenPlus       // +
	TokenMinus      // -
	TokenEquals     // =
	TokenNotEquals  // !=
	TokenLess       // <
	TokenLessEq     // <=
	TokenGreater    // >
	TokenGreaterEq  // >=
	TokenAnd        // and
	TokenOr         // or
	TokenMod        // mod
	TokenDiv        // div

	TokenLParen     // (
	TokenRParen     // )
	TokenLBracket   // [
	TokenRBracket   // ]
	TokenAt         // @
	TokenColonColon // ::
	TokenComma      // ,
	TokenDot        // .
	TokenDotDot     // ..
	TokenColon      // : (in QName prefix:local)
)

var tokenNames = map[TokenType]string{
	TokenEOF:         "EOF",
	TokenNumber:      "Number",
	TokenString:      "String",
	TokenName:        "Name",
	TokenStar:        "*",
	TokenVariableRef: "VariableRef",
	TokenSlash:       "/",
	TokenSlashSlash:  "//",
	TokenPipe:        "|",
	TokenPlus:        "+",
	TokenMinus:       "-",
	TokenEquals:      "=",
	TokenNotEquals:   "!=",
	TokenLess:        "<",
	TokenLessEq:      "<=",
	TokenGreater:     ">",
	TokenGreaterEq:   ">=",
	TokenAnd:         "and",
	TokenOr:          "or",
	TokenMod:         "mod",
	TokenDiv:         "div",
	TokenLParen:      "(",
	TokenRParen:      ")",
	TokenLBracket:    "[",
	TokenRBracket:    "]",
	TokenAt:          "@",
	TokenColonColon:  "::",
	TokenComma:       ",",
	TokenDot:         ".",
	TokenDotDot:      "..",
	TokenColon:       ":",
}

func (t TokenType) String() string {
	if s, ok := tokenNames[t]; ok {
		return s
	}
	return fmt.Sprintf("TokenType(%d)", int(t))
}

// Token is a single lexical token from an XPath expression.
type Token struct {
	Type  TokenType
	Value string
}

func (t Token) String() string {
	if t.Value != "" {
		return fmt.Sprintf("%s(%q)", t.Type, t.Value)
	}
	return t.Type.String()
}

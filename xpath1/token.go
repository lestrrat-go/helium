package xpath1

import "github.com/lestrrat-go/helium/internal/xpath1/lexer"

// TokenType identifies the kind of lexical token.
type TokenType = lexer.TokenType

// Token is a single lexical token from an XPath expression.
type Token = lexer.Token

// TokenEOF marks the end of input. TokenNumber, TokenString, TokenName,
// TokenStar, and TokenVariableRef represent literals and names.
// TokenSlash through TokenDiv represent operators.
// TokenLParen through TokenColon represent punctuation symbols.
const (
	TokenEOF         = lexer.TokenEOF
	TokenNumber      = lexer.TokenNumber
	TokenString      = lexer.TokenString
	TokenName        = lexer.TokenName
	TokenStar        = lexer.TokenStar
	TokenVariableRef = lexer.TokenVariableRef

	TokenSlash      = lexer.TokenSlash
	TokenSlashSlash = lexer.TokenSlashSlash
	TokenPipe       = lexer.TokenPipe
	TokenPlus       = lexer.TokenPlus
	TokenMinus      = lexer.TokenMinus
	TokenEquals     = lexer.TokenEquals
	TokenNotEquals  = lexer.TokenNotEquals
	TokenLess       = lexer.TokenLess
	TokenLessEq     = lexer.TokenLessEq
	TokenGreater    = lexer.TokenGreater
	TokenGreaterEq  = lexer.TokenGreaterEq
	TokenAnd        = lexer.TokenAnd
	TokenOr         = lexer.TokenOr
	TokenMod        = lexer.TokenMod
	TokenDiv        = lexer.TokenDiv

	TokenLParen     = lexer.TokenLParen
	TokenRParen     = lexer.TokenRParen
	TokenLBracket   = lexer.TokenLBracket
	TokenRBracket   = lexer.TokenRBracket
	TokenAt         = lexer.TokenAt
	TokenColonColon = lexer.TokenColonColon
	TokenComma      = lexer.TokenComma
	TokenDot        = lexer.TokenDot
	TokenDotDot     = lexer.TokenDotDot
	TokenColon      = lexer.TokenColon
)

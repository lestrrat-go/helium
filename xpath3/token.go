package xpath3

import "fmt"

// TokenType identifies the kind of lexical token.
type TokenType int

// Token constants. The first group (EOF through Div) is shared with XPath 1.0.
// The second group adds XPath 3.1 operators, keywords, and punctuation.
const (
	TokenEOF         TokenType = iota // end of input
	TokenNumber                       // 42, 3.14, 1e2
	TokenString                       // "hello", 'hello'
	TokenName                         // NCName (foo, bar)
	TokenStar                         // * (wildcard name test or multiply)
	TokenVariableRef                  // $name

	// TokenSlash and the following are operators inherited from XPath 1.0.
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

	// TokenLParen and the following are punctuation inherited from XPath 1.0.
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

	// TokenConcat and the following are new operators in XPath 3.1.
	TokenConcat    // ||
	TokenBang      // ! (simple map)
	TokenArrow     // =>
	TokenHash      // # (named function ref)
	TokenQMark     // ? (lookup / partial application)
	TokenIdiv      // idiv (integer division)
	TokenIntersect // intersect
	TokenExcept    // except
	TokenTo        // to (range)
	TokenUnion     // union (keyword form of |)

	// TokenLBrace and the following are new punctuation in XPath 3.1.
	TokenLBrace // {
	TokenRBrace // }

	// TokenEq and the following are value comparison keywords.
	TokenEq // eq
	TokenNe // ne
	TokenLt // lt
	TokenLe // le
	TokenGt // gt
	TokenGe // ge

	// TokenFor and the following are FLWOR keywords.
	TokenFor        // for
	TokenLet        // let
	TokenIn         // in
	TokenReturn     // return
	TokenWhere      // where
	TokenOrderBy    // order (followed by "by")
	TokenBy         // by
	TokenAscending  // ascending
	TokenDescending // descending
	TokenStable     // stable

	// TokenSome and the following are quantified expression keywords.
	TokenSome      // some
	TokenEvery     // every
	TokenSatisfies // satisfies

	// TokenIf and the following are control flow keywords.
	TokenIf    // if
	TokenThen  // then
	TokenElse  // else
	TokenTry   // try
	TokenCatch // catch

	// TokenInstanceOf and the following are type expression keywords.
	TokenInstanceOf // instance (followed by "of")
	TokenOf         // of
	TokenCastAs     // cast (followed by "as")
	TokenCastableAs // castable (followed by "as")
	TokenTreatAs    // treat (followed by "as")
	TokenAs         // as

	// TokenFunction and the following are constructor / inline function keywords.
	TokenFunction // function
	TokenMap      // map
	TokenArray    // array

	// TokenIs and the following are node comparison operators.
	TokenIs      // is
	TokenNodePre // << (node precedes)
	TokenNodeFol // >> (node follows)
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
	TokenConcat:      "||",
	TokenBang:        "!",
	TokenArrow:       "=>",
	TokenHash:        "#",
	TokenQMark:       "?",
	TokenIdiv:        "idiv",
	TokenIntersect:   "intersect",
	TokenExcept:      "except",
	TokenTo:          "to",
	TokenUnion:       "union",
	TokenLBrace:      "{",
	TokenRBrace:      "}",
	TokenEq:          "eq",
	TokenNe:          "ne",
	TokenLt:          "lt",
	TokenLe:          "le",
	TokenGt:          "gt",
	TokenGe:          "ge",
	TokenFor:         "for",
	TokenLet:         "let",
	TokenIn:          "in",
	TokenReturn:      "return",
	TokenWhere:       "where",
	TokenOrderBy:     "order",
	TokenBy:          "by",
	TokenAscending:   "ascending",
	TokenDescending:  "descending",
	TokenStable:      "stable",
	TokenSome:        "some",
	TokenEvery:       "every",
	TokenSatisfies:   "satisfies",
	TokenIf:          "if",
	TokenThen:        "then",
	TokenElse:        "else",
	TokenTry:         "try",
	TokenCatch:       "catch",
	TokenInstanceOf:  "instance",
	TokenOf:          "of",
	TokenCastAs:      "cast",
	TokenCastableAs:  "castable",
	TokenTreatAs:     "treat",
	TokenAs:          "as",
	TokenFunction:    "function",
	TokenMap:         "map",
	TokenArray:       "array",
	TokenIs:          "is",
	TokenNodePre:     "<<",
	TokenNodeFol:     ">>",
}

func (t TokenType) String() string {
	if s, ok := tokenNames[t]; ok {
		return s
	}
	return fmt.Sprintf("TokenType(%d)", int(t))
}

// Token is a single lexical token from an XPath 3.1 expression.
type Token struct {
	Type        TokenType
	Value       string
	SpaceBefore bool // true when whitespace preceded this token
}

func (t Token) String() string {
	if t.Value != "" {
		return fmt.Sprintf("%s(%q)", t.Type, t.Value)
	}
	return t.Type.String()
}

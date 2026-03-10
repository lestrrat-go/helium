package xpath3

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// lexer tokenizes an XPath 3.1 expression string.
type lexer struct {
	input  string
	pos    int
	tokens []Token
	idx    int // read cursor into tokens
}

// newLexer creates a lexer and tokenizes the entire input.
func newLexer(input string) (*lexer, error) {
	l := &lexer{input: input}
	if err := l.tokenize(); err != nil {
		return nil, err
	}
	return l, nil
}

// Next returns the next token and advances the cursor.
func (l *lexer) Next() Token {
	if l.idx >= len(l.tokens) {
		return Token{Type: TokenEOF}
	}
	t := l.tokens[l.idx]
	l.idx++
	return t
}

// Peek returns the next token without advancing.
func (l *lexer) Peek() Token {
	if l.idx >= len(l.tokens) {
		return Token{Type: TokenEOF}
	}
	return l.tokens[l.idx]
}

// PeekAt returns the token at offset positions ahead of the cursor (0 = current).
func (l *lexer) PeekAt(offset int) Token {
	i := l.idx + offset
	if i >= len(l.tokens) {
		return Token{Type: TokenEOF}
	}
	return l.tokens[i]
}

// Backup moves the cursor back by one token.
func (l *lexer) Backup() {
	if l.idx > 0 {
		l.idx--
	}
}

// Tokens returns all tokens (for testing).
func (l *lexer) Tokens() []Token {
	return l.tokens
}

// operatorKeywords maps keyword strings to their token types.
// These are recognized as keywords ONLY in operator context.
var operatorKeywords = map[string]TokenType{
	"and":        TokenAnd,
	"or":         TokenOr,
	"div":        TokenDiv,
	"mod":        TokenMod,
	"idiv":       TokenIdiv,
	"intersect":  TokenIntersect,
	"except":     TokenExcept,
	"to":         TokenTo,
	"eq":         TokenEq,
	"ne":         TokenNe,
	"lt":         TokenLt,
	"le":         TokenLe,
	"gt":         TokenGt,
	"ge":         TokenGe,
	"union":      TokenUnion,
	"instance":   TokenInstanceOf,
	"cast":       TokenCastAs,
	"castable":   TokenCastableAs,
	"treat":      TokenTreatAs,
	"as":         TokenAs,
	"in":         TokenIn,
	"return":     TokenReturn,
	"satisfies":  TokenSatisfies,
	"then":       TokenThen,
	"else":       TokenElse,
	"where":      TokenWhere,
	"by":         TokenBy,
	"ascending":  TokenAscending,
	"descending": TokenDescending,
	"stable":     TokenStable,
	"of":         TokenOf,
	"is":         TokenIs,
}

// alwaysKeywords are recognized regardless of context because they appear
// only at the start of an expression or sub-expression.
var alwaysKeywords = map[string]TokenType{
	"for":      TokenFor,
	"let":      TokenLet,
	"some":     TokenSome,
	"every":    TokenEvery,
	"if":       TokenIf,
	"try":      TokenTry,
	"catch":    TokenCatch,
	"function": TokenFunction,
	"map":      TokenMap,
	"array":    TokenArray,
}

func (l *lexer) tokenize() error {
	for l.pos < len(l.input) {
		l.skipWhitespace()
		if l.pos >= len(l.input) {
			break
		}

		r := l.peekRune()

		switch {
		case r == '(':
			l.emit(TokenLParen, "(")
			l.advanceRune(r)
		case r == ')':
			l.emit(TokenRParen, ")")
			l.advanceRune(r)
		case r == '[':
			l.emit(TokenLBracket, "[")
			l.advanceRune(r)
		case r == ']':
			l.emit(TokenRBracket, "]")
			l.advanceRune(r)
		case r == '{':
			l.emit(TokenLBrace, "{")
			l.advanceRune(r)
		case r == '}':
			l.emit(TokenRBrace, "}")
			l.advanceRune(r)
		case r == '@':
			l.emit(TokenAt, "@")
			l.advanceRune(r)
		case r == ',':
			l.emit(TokenComma, ",")
			l.advanceRune(r)
		case r == '+':
			l.emit(TokenPlus, "+")
			l.advanceRune(r)
		case r == '#':
			l.emit(TokenHash, "#")
			l.advanceRune(r)
		case r == '?':
			l.emit(TokenQMark, "?")
			l.advanceRune(r)
		case r == '|':
			l.advanceRune(r)
			if l.pos < len(l.input) && l.input[l.pos] == '|' {
				l.emit(TokenConcat, "||")
				l.pos++
			} else {
				l.emit(TokenPipe, "|")
			}
		case r == '=':
			l.advanceRune(r)
			if l.pos < len(l.input) && l.input[l.pos] == '>' {
				l.emit(TokenArrow, "=>")
				l.pos++
			} else {
				l.emit(TokenEquals, "=")
			}
		case r == '!':
			l.advanceRune(r)
			if l.pos < len(l.input) && l.input[l.pos] == '=' {
				l.emit(TokenNotEquals, "!=")
				l.pos++
			} else {
				l.emit(TokenBang, "!")
			}
		case r == '<':
			l.advanceRune(r)
			if l.pos < len(l.input) && l.input[l.pos] == '<' {
				l.emit(TokenNodePre, "<<")
				l.pos++
			} else if l.pos < len(l.input) && l.input[l.pos] == '=' {
				l.emit(TokenLessEq, "<=")
				l.pos++
			} else {
				l.emit(TokenLess, "<")
			}
		case r == '>':
			l.advanceRune(r)
			if l.pos < len(l.input) && l.input[l.pos] == '>' {
				l.emit(TokenNodeFol, ">>")
				l.pos++
			} else if l.pos < len(l.input) && l.input[l.pos] == '=' {
				l.emit(TokenGreaterEq, ">=")
				l.pos++
			} else {
				l.emit(TokenGreater, ">")
			}
		case r == '/':
			l.advanceRune(r)
			if l.pos < len(l.input) && l.input[l.pos] == '/' {
				l.emit(TokenSlashSlash, "//")
				l.pos++
			} else {
				l.emit(TokenSlash, "/")
			}
		case r == '.':
			l.advanceRune(r)
			switch {
			case l.pos < len(l.input) && l.input[l.pos] == '.':
				l.emit(TokenDotDot, "..")
				l.pos++
			case l.pos < len(l.input) && l.input[l.pos] >= '0' && l.input[l.pos] <= '9':
				// .5 style number
				l.pos-- // back to the dot
				l.scanNumber()
			default:
				l.emit(TokenDot, ".")
			}
		case r == '-':
			l.emit(TokenMinus, "-")
			l.advanceRune(r)
		case r == ':':
			l.advanceRune(r)
			if l.pos < len(l.input) && l.input[l.pos] == ':' {
				l.emit(TokenColonColon, "::")
				l.pos++
			} else {
				l.emit(TokenColon, ":")
			}
		case r == '*':
			l.emit(TokenStar, "*")
			l.advanceRune(r)
		case r == '$':
			l.advanceRune(r)
			name := l.scanNCName()
			if name == "" {
				return fmt.Errorf("%w: variable name after '$' at position %d", ErrUnexpectedToken, l.pos)
			}
			// Check for QName (prefix:local) — e.g., $err:code
			if l.pos < len(l.input) && l.input[l.pos] == ':' && l.pos+1 < len(l.input) {
				r2, _ := utf8.DecodeRuneInString(l.input[l.pos+1:])
				if isNCNameStart(r2) {
					l.pos++ // consume ':'
					local := l.scanNCName()
					name = name + ":" + local
				}
			}
			l.emit(TokenVariableRef, name)
		case r == '"' || r == '\'':
			s, err := l.scanString()
			if err != nil {
				return err
			}
			l.emit(TokenString, s)
		case r >= '0' && r <= '9':
			l.scanNumber()
		case isNCNameStart(r):
			l.scanNameOrKeyword()
		default:
			return fmt.Errorf("%w: %q at position %d", ErrUnexpectedChar, string(r), l.pos)
		}
	}
	return nil
}

func (l *lexer) peekRune() rune {
	r, _ := utf8.DecodeRuneInString(l.input[l.pos:])
	return r
}

func (l *lexer) advanceRune(r rune) {
	l.pos += utf8.RuneLen(r)
}

func (l *lexer) emit(typ TokenType, value string) {
	l.tokens = append(l.tokens, Token{Type: typ, Value: value})
}

func (l *lexer) skipWhitespace() {
	for l.pos < len(l.input) {
		r, size := utf8.DecodeRuneInString(l.input[l.pos:])
		if !unicode.IsSpace(r) {
			break
		}
		l.pos += size
	}
}

func (l *lexer) scanNCName() string {
	start := l.pos
	for l.pos < len(l.input) {
		r, size := utf8.DecodeRuneInString(l.input[l.pos:])
		if l.pos == start {
			if !isNCNameStart(r) {
				return ""
			}
		} else if !isNCNameChar(r) {
			break
		}
		l.pos += size
	}
	return l.input[start:l.pos]
}

func (l *lexer) scanString() (string, error) {
	quote := l.input[l.pos]
	l.pos++
	start := l.pos
	for l.pos < len(l.input) {
		if l.input[l.pos] == quote {
			// XPath 3.1: doubled quote is an escape ('' → ', "" → ")
			if l.pos+1 < len(l.input) && l.input[l.pos+1] == quote {
				l.pos += 2
				continue
			}
			s := l.input[start:l.pos]
			l.pos++
			// Unescape doubled quotes
			esc := string(quote) + string(quote)
			if strings.Contains(s, esc) {
				s = strings.ReplaceAll(s, esc, string(quote))
			}
			return s, nil
		}
		l.pos++
	}
	return "", fmt.Errorf("%w: starting at position %d", ErrUnterminatedString, start-1)
}

func (l *lexer) scanNumber() {
	start := l.pos
	for l.pos < len(l.input) && l.input[l.pos] >= '0' && l.input[l.pos] <= '9' {
		l.pos++
	}
	if l.pos < len(l.input) && l.input[l.pos] == '.' {
		l.pos++
		for l.pos < len(l.input) && l.input[l.pos] >= '0' && l.input[l.pos] <= '9' {
			l.pos++
		}
	}
	// Scientific notation: e.g. 1e2, 1.5E-3
	if l.pos < len(l.input) && (l.input[l.pos] == 'e' || l.input[l.pos] == 'E') {
		l.pos++
		if l.pos < len(l.input) && (l.input[l.pos] == '+' || l.input[l.pos] == '-') {
			l.pos++
		}
		for l.pos < len(l.input) && l.input[l.pos] >= '0' && l.input[l.pos] <= '9' {
			l.pos++
		}
	}
	l.emit(TokenNumber, l.input[start:l.pos])
}

// scanNameOrKeyword scans an NCName and determines if it's a keyword or name.
// Colon handling is done in the main tokenize loop, not here.
func (l *lexer) scanNameOrKeyword() {
	name := l.scanNCName()

	// URIQualifiedName: Q{uri}local
	if name == "Q" && l.pos < len(l.input) && l.input[l.pos] == '{' {
		l.pos++ // consume '{'
		braceStart := l.pos
		for l.pos < len(l.input) && l.input[l.pos] != '}' {
			l.pos++
		}
		if l.pos >= len(l.input) {
			// Unterminated — emit what we have; parser will error
			l.emit(TokenName, name)
			return
		}
		uri := l.input[braceStart:l.pos]
		l.pos++ // consume '}'
		local := l.scanNCName()
		l.emit(TokenName, "Q{"+uri+"}"+local)
		return
	}

	// Disambiguate operator keywords from names.
	// Per XPath spec: keywords are operators ONLY when preceded by
	// a value-producing token (where an operator is expected).
	if l.isOperatorContext() {
		if tokType, ok := operatorKeywords[name]; ok {
			l.emit(tokType, name)
			return
		}
	}

	// Always-keywords: for, let, some, every, if, try, catch, function, map, array
	// These start sub-expressions, so they are recognized regardless of context.
	// However, if followed by '(' they might be function calls — parser handles this.
	if tokType, ok := alwaysKeywords[name]; ok {
		l.emit(tokType, name)
		return
	}

	l.emit(TokenName, name)
}

// isOperatorContext returns true if an operator keyword is expected at
// this point. Per the XPath spec disambiguation rules, an operator is
// expected when the preceding token is a value-producing token.
func (l *lexer) isOperatorContext() bool {
	if len(l.tokens) == 0 {
		return false
	}
	prev := l.tokens[len(l.tokens)-1]
	switch prev.Type {
	// Value-producing tokens: an operator is expected after these.
	case TokenName, TokenNumber, TokenString, TokenRParen, TokenRBracket,
		TokenDot, TokenDotDot, TokenStar, TokenVariableRef, TokenRBrace, TokenQMark:
		return true
	// Multi-token keyword pairs: "instance of", "cast as", "castable as",
	// "treat as" — the secondary keyword follows the primary keyword.
	case TokenInstanceOf, TokenCastAs, TokenCastableAs, TokenTreatAs:
		return true
	// "for $x in", "let $x := ... return", "where ... order by",
	// "some/every $x in ... satisfies", "if (...) then ... else"
	// — these keywords precede other keywords in their clauses.
	case TokenFor, TokenLet, TokenSome, TokenEvery:
		return true
	}
	return false
}

// PrettyTokens returns a human-readable representation of all tokens.
func (l *lexer) PrettyTokens() string {
	var b strings.Builder
	for i, t := range l.tokens {
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString(t.String())
	}
	return b.String()
}

// isNCNameStart returns true if r is a valid start of an NCName.
func isNCNameStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

// isNCNameChar returns true if r is a valid NCName continuation character.
func isNCNameChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) ||
		r == '.' || r == '-' || r == '_'
}

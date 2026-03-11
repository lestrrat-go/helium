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
			// Check for XPath comment (: ... :)
			if l.pos+1 < len(l.input) && l.input[l.pos+1] == ':' {
				if err := l.skipComment(); err != nil {
					return err
				}
			} else {
				l.emit(TokenLParen, "(")
				l.advanceRune(r)
			}
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
				if err := l.scanNumber(); err != nil {
					return err
				}
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
			// URIQualifiedName: $Q{uri}local
			if name == "Q" && l.pos < len(l.input) && l.input[l.pos] == '{' {
				l.pos++ // consume '{'
				braceStart := l.pos
				for l.pos < len(l.input) && l.input[l.pos] != '}' {
					l.pos++
				}
				if l.pos < len(l.input) {
					uri := strings.TrimSpace(l.input[braceStart:l.pos])
					l.pos++ // consume '}'
					local := l.scanNCName()
					if uri == "" {
						// Q{}local with empty URI is equivalent to unprefixed local
						name = local
					} else {
						name = "Q{" + uri + "}" + local
					}
				}
			} else if l.pos < len(l.input) && l.input[l.pos] == ':' && l.pos+1 < len(l.input) {
				// Check for QName (prefix:local) — e.g., $err:code
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
			if err := l.scanNumber(); err != nil {
				return err
			}
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

// skipComment skips an XPath comment delimited by (: and :).
// The lexer position must be at the opening '('. Nested comments are supported.
func (l *lexer) skipComment() error {
	start := l.pos
	l.pos += 2 // skip "(:"
	depth := 1
	for l.pos < len(l.input) && depth > 0 {
		if l.pos+1 < len(l.input) && l.input[l.pos] == '(' && l.input[l.pos+1] == ':' {
			depth++
			l.pos += 2
		} else if l.pos+1 < len(l.input) && l.input[l.pos] == ':' && l.input[l.pos+1] == ')' {
			depth--
			l.pos += 2
		} else {
			l.pos++
		}
	}
	if depth > 0 {
		return fmt.Errorf("%w: unterminated comment starting at position %d", ErrUnexpectedToken, start)
	}
	return nil
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

func (l *lexer) scanNumber() error {
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
	// XPath 3.1 A.2.1: a numeric literal immediately followed by a name start
	// character (letter, underscore) is a lexical error.
	if l.pos < len(l.input) {
		r, _ := utf8.DecodeRuneInString(l.input[l.pos:])
		if isNCNameStart(r) {
			return fmt.Errorf("%w: numeric literal immediately followed by name at position %d", ErrUnexpectedToken, start)
		}
	}
	l.emit(TokenNumber, l.input[start:l.pos])
	return nil
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
		uri := strings.TrimSpace(l.input[braceStart:l.pos])
		l.pos++ // consume '}'
		local := l.scanNCName()
		if uri == "" {
			l.emit(TokenName, local)
		} else {
			l.emit(TokenName, "Q{"+uri+"}"+local)
		}
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
	// These start sub-expressions, so they are recognized regardless of context —
	// EXCEPT after '/' or '//' where they must be treated as element name tests,
	// unless immediately followed by '{' (curly array/map constructor).
	if tokType, ok := alwaysKeywords[name]; ok {
		if !l.isAfterSlash() {
			l.emit(tokType, name)
			return
		}
		// After '/', array{ and map{ are still constructors, not name tests
		if (tokType == TokenArray || tokType == TokenMap) && l.pos < len(l.input) && l.input[l.pos] == '{' {
			l.emit(tokType, name)
			return
		}
	}

	l.emit(TokenName, name)
}

// isAfterSlash returns true if the previous token is '/' or '//'.
// In this context, keywords must be treated as element name tests.
func (l *lexer) isAfterSlash() bool {
	if len(l.tokens) == 0 {
		return false
	}
	prev := l.tokens[len(l.tokens)-1]
	return prev.Type == TokenSlash || prev.Type == TokenSlashSlash
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
	// Note: TokenQMark is NOT here — after '?' we expect a lookup key (NCName),
	// so keywords like 'or', 'and' must be treated as names, not operators.
	case TokenName, TokenNumber, TokenString, TokenRParen, TokenRBracket,
		TokenDot, TokenDotDot, TokenStar, TokenVariableRef, TokenRBrace:
		return true
	case TokenQMark:
		// '?' is value-producing when it follows a type name (occurrence indicator
		// in "cast as xs:double ?", "instance of xs:integer ?"), but NOT when it
		// starts a unary lookup expression ("?key"). Check second-to-last token:
		// if it's a name, this '?' is an occurrence indicator.
		if len(l.tokens) >= 2 {
			prev2 := l.tokens[len(l.tokens)-2]
			if prev2.Type == TokenName {
				return true
			}
		}
		return false
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

// isValidNCName checks whether s is a valid XML NCName (non-empty, starts with
// a valid NCNameStart character, and all subsequent characters are NCNameChar).
func isValidNCName(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !isNCNameStart(r) {
				return false
			}
		} else {
			if !isNCNameChar(r) {
				return false
			}
		}
	}
	return true
}

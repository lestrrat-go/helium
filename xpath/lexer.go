package xpath

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Lexer tokenizes an XPath expression string.
type Lexer struct {
	input  string
	pos    int
	tokens []Token
	idx    int // read cursor into tokens
}

// NewLexer creates a lexer and tokenizes the entire input.
func NewLexer(input string) (*Lexer, error) {
	l := &Lexer{input: input}
	if err := l.tokenize(); err != nil {
		return nil, err
	}
	return l, nil
}

// Next returns the next token and advances the cursor.
func (l *Lexer) Next() Token {
	if l.idx >= len(l.tokens) {
		return Token{Type: TokenEOF}
	}
	t := l.tokens[l.idx]
	l.idx++
	return t
}

// Peek returns the next token without advancing.
func (l *Lexer) Peek() Token {
	if l.idx >= len(l.tokens) {
		return Token{Type: TokenEOF}
	}
	return l.tokens[l.idx]
}

// Backup moves the cursor back by one token.
func (l *Lexer) Backup() {
	if l.idx > 0 {
		l.idx--
	}
}

// Tokens returns all tokens (for testing).
func (l *Lexer) Tokens() []Token {
	return l.tokens
}

func (l *Lexer) tokenize() error {
	for l.pos < len(l.input) {
		l.skipWhitespace()
		if l.pos >= len(l.input) {
			break
		}

		r := l.peekRune()

		switch {
		case r == '(':
			l.emit(TokenLParen, "(")
			l.advance(1)
		case r == ')':
			l.emit(TokenRParen, ")")
			l.advance(1)
		case r == '[':
			l.emit(TokenLBracket, "[")
			l.advance(1)
		case r == ']':
			l.emit(TokenRBracket, "]")
			l.advance(1)
		case r == '@':
			l.emit(TokenAt, "@")
			l.advance(1)
		case r == ',':
			l.emit(TokenComma, ",")
			l.advance(1)
		case r == '+':
			l.emit(TokenPlus, "+")
			l.advance(1)
		case r == '|':
			l.emit(TokenPipe, "|")
			l.advance(1)
		case r == '=':
			l.emit(TokenEquals, "=")
			l.advance(1)
		case r == '!':
			l.advance(1)
			if l.pos < len(l.input) && l.input[l.pos] == '=' {
				l.emit(TokenNotEquals, "!=")
				l.advance(1)
			} else {
				return fmt.Errorf("unexpected '!' at position %d", l.pos-1)
			}
		case r == '<':
			l.advance(1)
			if l.pos < len(l.input) && l.input[l.pos] == '=' {
				l.emit(TokenLessEq, "<=")
				l.advance(1)
			} else {
				l.emit(TokenLess, "<")
			}
		case r == '>':
			l.advance(1)
			if l.pos < len(l.input) && l.input[l.pos] == '=' {
				l.emit(TokenGreaterEq, ">=")
				l.advance(1)
			} else {
				l.emit(TokenGreater, ">")
			}
		case r == '/':
			l.advance(1)
			if l.pos < len(l.input) && l.input[l.pos] == '/' {
				l.emit(TokenSlashSlash, "//")
				l.advance(1)
			} else {
				l.emit(TokenSlash, "/")
			}
		case r == '.':
			l.advance(1)
			if l.pos < len(l.input) && l.input[l.pos] == '.' {
				l.emit(TokenDotDot, "..")
				l.advance(1)
			} else if l.pos < len(l.input) && l.input[l.pos] >= '0' && l.input[l.pos] <= '9' {
				// .5 style number
				l.pos-- // back to the dot
				l.scanNumber()
			} else {
				l.emit(TokenDot, ".")
			}
		case r == '-':
			l.emit(TokenMinus, "-")
			l.advance(1)
		case r == '*':
			// * can be multiply operator or name test wildcard.
			// Disambiguated by the parser based on context,
			// but the lexer always emits TokenStar.
			l.emit(TokenStar, "*")
			l.advance(1)
		case r == '$':
			l.advance(1)
			name := l.scanNCName()
			if name == "" {
				return fmt.Errorf("expected variable name after '$' at position %d", l.pos)
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
			return fmt.Errorf("unexpected character %q at position %d", string(r), l.pos)
		}
	}
	return nil
}

func (l *Lexer) peekRune() rune {
	r, _ := utf8.DecodeRuneInString(l.input[l.pos:])
	return r
}

func (l *Lexer) advance(n int) {
	l.pos += n
}

func (l *Lexer) emit(typ TokenType, value string) {
	l.tokens = append(l.tokens, Token{Type: typ, Value: value})
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) {
		r, size := utf8.DecodeRuneInString(l.input[l.pos:])
		if !unicode.IsSpace(r) {
			break
		}
		l.pos += size
	}
}

func (l *Lexer) scanNCName() string {
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

func (l *Lexer) scanString() (string, error) {
	quote := l.input[l.pos]
	l.advance(1)
	start := l.pos
	for l.pos < len(l.input) {
		if l.input[l.pos] == quote {
			s := l.input[start:l.pos]
			l.advance(1)
			return s, nil
		}
		l.advance(1)
	}
	return "", fmt.Errorf("unterminated string starting at position %d", start-1)
}

func (l *Lexer) scanNumber() {
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
	l.emit(TokenNumber, l.input[start:l.pos])
}

// scanNameOrKeyword scans an NCName and determines if it's a keyword
// (and, or, div, mod) or a regular name. It also handles the :: suffix
// for axis names and checks for a following '(' which indicates a
// function call or node type test.
func (l *Lexer) scanNameOrKeyword() {
	name := l.scanNCName()

	// Check for '::'  → axis name
	if l.pos+1 < len(l.input) && l.input[l.pos] == ':' && l.input[l.pos+1] == ':' {
		l.emit(TokenName, name)
		l.emit(TokenColonColon, "::")
		l.pos += 2
		return
	}

	// Check for ':'  → QName prefix
	if l.pos < len(l.input) && l.input[l.pos] == ':' {
		// Could be prefix:local or prefix:*
		l.emit(TokenName, name)
		l.emit(TokenColon, ":")
		l.pos++
		if l.pos < len(l.input) && l.input[l.pos] == '*' {
			l.emit(TokenStar, "*")
			l.pos++
		} else {
			local := l.scanNCName()
			if local != "" {
				l.emit(TokenName, local)
			}
		}
		return
	}

	// Disambiguate operator keywords from names.
	// Per XPath spec: "and", "or", "mod", "div" are operators ONLY when
	// the preceding token is such that an operator is expected.
	if l.isOperatorContext(name) {
		switch name {
		case "and":
			l.emit(TokenAnd, "and")
			return
		case "or":
			l.emit(TokenOr, "or")
			return
		case "mod":
			l.emit(TokenMod, "mod")
			return
		case "div":
			l.emit(TokenDiv, "div")
			return
		}
	}

	l.emit(TokenName, name)
}

// isOperatorContext returns true if an operator keyword is expected at
// this point. Per the XPath spec disambiguation rules, an operator is
// expected when the preceding token is a:
//   - Name that is NOT followed by '('
//   - Number
//   - Closing delimiter: ) ] or a NodeType/FunctionName token
//
// In other words, if the last token is a value-producing token, the next
// name should be parsed as an operator keyword.
func (l *Lexer) isOperatorContext(name string) bool {
	if name != "and" && name != "or" && name != "mod" && name != "div" {
		return false
	}
	if len(l.tokens) == 0 {
		return false
	}
	prev := l.tokens[len(l.tokens)-1]
	switch prev.Type {
	case TokenName, TokenNumber, TokenString, TokenRParen, TokenRBracket, TokenDot, TokenDotDot, TokenStar, TokenVariableRef:
		return true
	}
	return false
}

// isNCNameStart returns true if r is a valid start of an NCName
// (XML NameStartChar minus colon).
func isNCNameStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

// isNCNameChar returns true if r is a valid NCName continuation character.
func isNCNameChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) ||
		r == '.' || r == '-' || r == '_' ||
		unicode.In(r, unicode.Extender)
}

// PrettyTokens returns a human-readable representation of all tokens.
func (l *Lexer) PrettyTokens() string {
	var b strings.Builder
	for i, t := range l.tokens {
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString(t.String())
	}
	return b.String()
}

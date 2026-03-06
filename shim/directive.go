package shim

import (
	"bytes"
	"io"
)

// scanProlog reads from r until the first element start tag, tokenizing
// the XML prolog. It extracts CharData (whitespace), ProcInst, Comment,
// and Directive tokens. The SAX parser does not emit any of these for the
// prolog portion, so we handle them here and let SAX take over from the
// first element.
//
// Returns:
//   - tokens: the prolog tokens in order
//   - combined: a reader that replays everything read plus the remaining input
//     (so the SAX parser still sees the full document including the prolog)
//   - prologOnly: true if the entire input is prolog (no root element found)
func scanProlog(r io.Reader) ([]Token, io.Reader, bool, error) {
	s := &prologScanner{r: r}
	tokens, err := s.scan()
	prologOnly := (err == io.EOF)
	if err != nil && err != io.EOF {
		return nil, nil, false, err
	}

	// If the prolog contained an XML declaration, blank it out in the
	// replay buffer so the SAX parser doesn't reject it as a misplaced
	// PI (the parser errors on <?xml?> when preceded by whitespace).
	// Replacing with spaces preserves byte offsets for InputOffset().
	if s.xmlDeclEnd > s.xmlDeclStart {
		buf := s.buf.Bytes()
		for i := s.xmlDeclStart; i < s.xmlDeclEnd && i < len(buf); i++ {
			buf[i] = ' '
		}
	}

	combined := io.MultiReader(bytes.NewReader(s.buf.Bytes()), r)
	return tokens, combined, prologOnly, nil
}

type prologScanner struct {
	r            io.Reader
	buf          bytes.Buffer // all bytes read from r, for replay
	peek         []byte       // ungotten bytes
	xmlDeclStart int          // byte offset of '<' in <?xml ...?>
	xmlDeclEnd   int          // byte offset after '>' in <?xml ...?>
}

func (s *prologScanner) readByte() (byte, error) {
	if len(s.peek) > 0 {
		b := s.peek[len(s.peek)-1]
		s.peek = s.peek[:len(s.peek)-1]
		// Don't write to buf — byte is already there from initial read
		return b, nil
	}
	var tmp [1]byte
	_, err := s.r.Read(tmp[:])
	if err != nil {
		return 0, err
	}
	s.buf.WriteByte(tmp[0])
	return tmp[0], nil
}

func (s *prologScanner) unreadByte(b byte) {
	// Push byte back for re-reading. Don't modify buf — the byte is
	// already in buf from when it was originally read from r.
	s.peek = append(s.peek, b)
}

func (s *prologScanner) scan() ([]Token, error) {
	var tokens []Token
	var wsAccum bytes.Buffer

	flushWS := func() {
		if wsAccum.Len() > 0 {
			tokens = append(tokens, CharData(append([]byte(nil), wsAccum.Bytes()...)))
			wsAccum.Reset()
		}
	}

	for {
		b, err := s.readByte()
		if err != nil {
			flushWS()
			return tokens, err
		}

		if isWhitespace(b) {
			wsAccum.WriteByte(b)
			continue
		}

		if b != '<' {
			// Non-whitespace, non-'<' — shouldn't happen in prolog.
			// Put it back and let SAX handle it.
			s.unreadByte(b)
			flushWS()
			return tokens, nil
		}

		// We have '<'. Peek at next byte.
		b2, err := s.readByte()
		if err != nil {
			// Incomplete — flush what we have
			flushWS()
			return tokens, err
		}

		switch {
		case b2 == '?':
			// Processing instruction <?...?>
			flushWS()
			// Record position before scanning PI body. The '<' and '?'
			// are already in buf, so the PI starts 2 bytes back.
			piStart := s.buf.Len() - 2
			tok, err := s.scanPI()
			if err != nil {
				return tokens, err
			}
			if pi, ok := tok.(ProcInst); ok && pi.Target == "xml" {
				s.xmlDeclStart = piStart
				s.xmlDeclEnd = s.buf.Len()
			}
			tokens = append(tokens, tok)

		case b2 == '!':
			// Could be comment or directive. Peek more.
			b3, err := s.readByte()
			if err != nil {
				flushWS()
				return tokens, err
			}

			if b3 == '-' {
				b4, err := s.readByte()
				if err != nil {
					flushWS()
					return tokens, err
				}
				if b4 == '-' {
					// Comment <!--...-->
					flushWS()
					tok, err := s.scanComment()
					if err != nil {
						return tokens, err
					}
					tokens = append(tokens, tok)
				} else {
					// <!-X — treat as directive start
					s.unreadByte(b4)
					s.unreadByte(b3)
					flushWS()
					tok, err := s.scanDirective()
					if err != nil {
						return tokens, err
					}
					tokens = append(tokens, tok)
				}
			} else if b3 == '[' {
				// <![CDATA[ — shouldn't be in prolog. Put back and let SAX handle.
				s.unreadByte(b3)
				s.unreadByte(b2)
				s.unreadByte(b)
				flushWS()
				return tokens, nil
			} else {
				// Directive: <!DOCTYPE ...>, etc. Put b3 back so scanDirective sees it.
				s.unreadByte(b3)
				flushWS()
				tok, err := s.scanDirective()
				if err != nil {
					return tokens, err
				}
				tokens = append(tokens, tok)
			}

		default:
			// Element start tag like <name. Put both bytes back.
			s.unreadByte(b2)
			s.unreadByte(b)
			flushWS()
			return tokens, nil
		}
	}
}

// scanPI reads a processing instruction after '<?' has been consumed.
// Returns a ProcInst token.
func (s *prologScanner) scanPI() (Token, error) {
	var body bytes.Buffer
	var prev byte
	for {
		b, err := s.readByte()
		if err != nil {
			return ProcInst{Target: body.String()}, err
		}
		if prev == '?' && b == '>' {
			// Remove trailing '?' from body
			raw := body.Bytes()
			raw = raw[:len(raw)-1]

			// Split into target and data
			target, data := splitPI(raw)
			return ProcInst{Target: target, Inst: data}, nil
		}
		body.WriteByte(b)
		prev = b
	}
}

// splitPI splits "target data" into target and data parts.
func splitPI(raw []byte) (string, []byte) {
	i := 0
	for i < len(raw) && !isWhitespace(raw[i]) {
		i++
	}
	target := string(raw[:i])
	if i >= len(raw) {
		return target, nil
	}
	// Skip whitespace between target and data
	for i < len(raw) && isWhitespace(raw[i]) {
		i++
	}
	if i >= len(raw) {
		return target, nil
	}
	return target, raw[i:]
}

// scanComment reads a comment after '<!--' has been consumed.
// Returns a Comment token containing the text between <!-- and -->.
func (s *prologScanner) scanComment() (Token, error) {
	var body bytes.Buffer
	var b0, b1 byte
	for {
		b, err := s.readByte()
		if err != nil {
			return Comment(body.Bytes()), err
		}
		if b0 == '-' && b1 == '-' && b == '>' {
			// Remove trailing "--" from body
			raw := body.Bytes()
			raw = raw[:len(raw)-2]
			return Comment(append([]byte(nil), raw...)), nil
		}
		body.WriteByte(b)
		b0, b1 = b1, b
	}
}

// scanDirective reads a directive after '<!' has been consumed.
// It handles the content between <! and the matching >, including nested
// angle brackets, quoted strings, and comments (which are replaced with spaces).
// Returns a Directive token.
//
// This mirrors the algorithm in encoding/xml's rawToken method, including
// the "goto HandleB" pattern where a non-matching byte after a failed
// comment check is processed through the main switch.
func (s *prologScanner) scanDirective() (Token, error) {
	var content bytes.Buffer
	inquote := byte(0)
	depth := 0

	for {
		b, err := s.readByte()
		if err != nil {
			return Directive(content.Bytes()), err
		}

		if inquote == 0 && b == '>' && depth == 0 {
			return Directive(append([]byte(nil), content.Bytes()...)), nil
		}

		// handleB: process byte b through the main switch.
		// This label-equivalent is needed because failed comment checks
		// need to re-enter the switch with the non-matching byte.
	handleB:
		content.WriteByte(b)

		switch {
		case b == inquote:
			inquote = 0

		case inquote != 0:
			// Inside quotes, no special handling

		case b == '\'' || b == '"':
			inquote = b

		case b == '>' && inquote == 0:
			depth--

		case b == '<' && inquote == 0:
			// Look for <!-- to begin a comment.
			match := "!--"
			matched := true
			var i int
			for i = 0; i < len(match); i++ {
				nb, err := s.readByte()
				if err != nil {
					for j := 0; j < i; j++ {
						content.WriteByte(match[j])
					}
					return Directive(content.Bytes()), err
				}
				if nb != match[i] {
					// Not a comment. Write prefix, then process nb through handleB.
					for j := 0; j < i; j++ {
						content.WriteByte(match[j])
					}
					depth++
					b = nb
					matched = false
					break
				}
			}

			if !matched {
				goto handleB
			}

			// Successfully matched <!--. Remove the '<' we just wrote.
			content.Truncate(content.Len() - 1)

			// Read until -->
			var b0, b1 byte
			for {
				cb, err := s.readByte()
				if err != nil {
					return Directive(content.Bytes()), err
				}
				if b0 == '-' && b1 == '-' && cb == '>' {
					break
				}
				b0, b1 = b1, cb
			}
			// Replace the comment with a space
			content.WriteByte(' ')

		default:
			// Already written via handleB
		}
	}
}

func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

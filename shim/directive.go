package shim

import (
	"bytes"
	stdxml "encoding/xml"
	"io"

	helium "github.com/lestrrat-go/helium"
)

// maxPrologSize bounds the number of bytes the prolog scanner will buffer
// before reaching the first element start tag. Without it, a document with a
// huge prolog (megabytes of comments or a giant internal DTD subset) ahead of
// the root element forces unbounded buffering — a memory-amplification DoS on
// untrusted input. The cap mirrors helium.MaxExternalDTDSize (10 MiB), the
// parser's existing content-size bound.
const maxPrologSize = helium.MaxExternalDTDSize

// errPrologTooLarge is returned when the prolog exceeds maxPrologSize before
// the first element start tag is reached.
var errPrologTooLarge = &stdxml.SyntaxError{Msg: "prolog exceeds maximum size before root element"}

// scanProlog reads from r until the first element start tag, tokenizing
// the XML prolog. It extracts CharData (whitespace), ProcInst, Comment,
// and Directive tokens and delivers them itself, then blanks each token's
// bytes out of the replay buffer so the SAX parser re-emits none of them and
// SAX takes over cleanly from the first element.
//
// Returns:
//   - tokens: the prolog tokens in order
//   - combined: a reader that replays everything read plus the remaining input
//     (the prolog tokens blanked to whitespace, so the SAX parser sees only the
//     root element and everything after)
//   - prologOnly: true if the entire input is prolog (no root element found)
func scanProlog(r io.Reader) ([]Token, io.Reader, bool, error) {
	s := &prologScanner{r: r}
	tokens, err := s.scan()

	// A prolog-size overflow is reported via s.sizeErr regardless of how the
	// inner sub-scanners (PI/comment/directive) rewrote the read error, so
	// surface it first.
	if s.sizeErr != nil {
		return nil, nil, false, s.sizeErr
	}

	if err != nil && err != io.EOF {
		// Syntax error from prolog validation
		return nil, nil, false, err
	}

	prologOnly := (err == io.EOF)

	// Blank every scanned prolog token (the XML declaration, comments, PIs,
	// and directives) out of the replay buffer. scanProlog already delivered
	// them as prologTokens, so the SAX parser must not re-emit them. Whatever
	// whitespace lies between them is left untouched; blanking the tokens to
	// spaces makes the whole pre-root region whitespace, which helium's parser
	// ignores. This also keeps the parser from rejecting the declaration as a
	// misplaced PI (it errors on <?xml?> when preceded by whitespace).
	// Replacing with spaces preserves byte offsets for InputOffset().
	buf := s.buf.Bytes()
	for _, span := range s.blankSpans {
		for i := span[0]; i < span[1] && i < len(buf); i++ {
			buf[i] = ' '
		}
	}

	combined := io.MultiReader(bytes.NewReader(s.buf.Bytes()), r)
	return tokens, combined, prologOnly, nil
}

type prologScanner struct {
	r          io.Reader
	buf        bytes.Buffer // all bytes read from r, for replay
	peek       []byte       // ungotten bytes
	blankSpans [][2]int     // [start,end) byte ranges of prolog tokens to blank in the replay buffer
	sawContent bool         // whitespace, a PI, comment or directive has been scanned
	sizeErr    error        // set when the prolog exceeds maxPrologSize
}

// errDeclNotAtStart is returned when an XML declaration is preceded by anything
// other than whitespace.
var errDeclNotAtStart = &stdxml.SyntaxError{
	Msg: "XML declaration allowed only at the start of the document",
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
	if s.buf.Len() > maxPrologSize {
		s.sizeErr = errPrologTooLarge
		return 0, errPrologTooLarge
	}
	return tmp[0], nil
}

func (s *prologScanner) unreadByte(b byte) {
	// Push byte back for re-reading. Don't modify buf — the byte is
	// already in buf from when it was originally read from r.
	s.peek = append(s.peek, b)
}

var errUnexpectedEOF = &stdxml.SyntaxError{Msg: "unexpected EOF"}

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
			return tokens, err // clean EOF at top level
		}

		if isWhitespace(b) {
			// Whitespace ahead of a later "<?xml" declaration displaces it from
			// the start of the document: a declaration is legal only at position 0
			// (prolog ::= XMLDecl? Misc* ...), so any whitespace before it is
			// content for the placement rule and makes the declaration misplaced.
			// It does not displace a later root ELEMENT — that path never reaches
			// the reserved-target check below. A leading byte-order mark is not
			// whitespace and never reaches here: it stops the scan (a non-'<',
			// non-whitespace byte), leaving helium to judge the BOM+declaration in
			// context, so the BOM stays exempt.
			s.sawContent = true
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
			// Incomplete '<' at EOF
			flushWS()
			return tokens, errUnexpectedEOF
		}

		switch b2 {
		case '?':
			// Processing instruction <?...?>
			flushWS()
			// Record position before scanning PI body. The '<' and '?'
			// are already in buf, so the PI starts 2 bytes back.
			piStart := s.buf.Len() - 2
			tok, err := s.scanPI()
			if err != nil {
				return tokens, errUnexpectedEOF
			}
			pi := tok.(ProcInst) //nolint:forcetypeassert

			// Validate PI target
			if pi.Target == "" {
				return tokens, &stdxml.SyntaxError{Msg: "expected target name after <?"}
			}
			if !isXMLName(pi.Target) {
				return tokens, &stdxml.SyntaxError{Msg: "expected target name after <?"}
			}

			// Blank this PI (including the declaration) out of the replay
			// buffer so the SAX parser never re-emits what prologTokens
			// already delivered.
			s.blankSpans = append(s.blankSpans, [2]int{piStart, s.buf.Len()})

			if isReservedXMLTarget(pi.Target) {
				// A target equal to "xml" in any casing is the reserved name
				// (XML 1.0 §2.6). XMLDecl ::= '<?xml' ... is the FIRST thing in a
				// document (prolog ::= XMLDecl? Misc* ...), so anything scanned
				// ahead of it — leading whitespace, an earlier declaration, a
				// comment, a PI, a doctype — makes it a misplaced PI, matching
				// helium's own verdict for a declaration not at position 0. The
				// bytes are blanked out of the replay buffer so helium never
				// re-parses them; the drained ProcInst is then held to
				// checkXMLDecl (in readToken), which accepts the lowercase "xml"
				// as a declaration and rejects any other casing as an illegal
				// target.
				if s.sawContent {
					return tokens, errDeclNotAtStart
				}
			}
			s.sawContent = true
			tokens = append(tokens, tok)

		case '!':
			// Could be comment or directive. Peek more.
			b3, err := s.readByte()
			if err != nil {
				flushWS()
				return tokens, errUnexpectedEOF
			}

			switch b3 {
			case '-':
				b4, err := s.readByte()
				if err != nil {
					flushWS()
					return tokens, errUnexpectedEOF
				}
				if b4 == '-' {
					// Comment <!--...-->
					flushWS()
					// The '<', '!', '-', '-' are already in buf, so the
					// comment starts 4 bytes back.
					commentStart := s.buf.Len() - 4
					tok, err := s.scanComment()
					if err != nil {
						return tokens, errUnexpectedEOF
					}
					s.blankSpans = append(s.blankSpans, [2]int{commentStart, s.buf.Len()})
					s.sawContent = true
					tokens = append(tokens, tok)
				} else {
					// <!-X where X != '-' → invalid
					return tokens, &stdxml.SyntaxError{
						Msg: "invalid sequence <!- not part of <!--",
					}
				}
			case '[':
				// <![ in prolog is invalid
				return tokens, &stdxml.SyntaxError{Msg: "invalid <![ sequence"}
			default:
				// Directive: <!DOCTYPE ...>, etc. Put b3 back so scanDirective sees it.
				s.unreadByte(b3)
				flushWS()
				// The '<', '!', and b3 (unread but retained) are already in
				// buf, so the directive starts 3 bytes back.
				directiveStart := s.buf.Len() - 3
				tok, err := s.scanDirective()
				if err != nil {
					return tokens, errUnexpectedEOF
				}
				s.blankSpans = append(s.blankSpans, [2]int{directiveStart, s.buf.Len()})
				s.sawContent = true
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
			for i = range len(match) {
				nb, err := s.readByte()
				if err != nil {
					for j := range i {
						content.WriteByte(match[j])
					}
					return Directive(content.Bytes()), err
				}
				if nb != match[i] {
					// Not a comment. Write prefix, then process nb through handleB.
					for j := range i {
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

// isLeadingBOM reports whether tok is CharData consisting solely of byte-order
// marks (U+FEFF). A leading BOM is document framing, not content, so it does not
// displace a later XML declaration from the start of the document. Leading
// whitespace, by contrast, DOES count as content: an XML declaration is legal
// only at document position 0 (prolog ::= XMLDecl? Misc* ...), so any whitespace
// ahead of it makes it misplaced, matching helium's verdict on the byte paths.
// Every other token, and any CharData carrying a non-BOM character, counts as
// content.
func isLeadingBOM(tok Token) bool {
	cd, ok := tok.(CharData)
	if !ok {
		return false
	}
	if len(cd) == 0 {
		return false
	}
	for _, r := range string(cd) {
		if r != '\uFEFF' {
			return false
		}
	}
	return true
}

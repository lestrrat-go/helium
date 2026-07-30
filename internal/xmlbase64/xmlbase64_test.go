package xmlbase64_test

import (
	"encoding/base64"
	"errors"
	"math/rand"
	"runtime"
	"slices"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/xmlbase64"
	"github.com/stretchr/testify/require"
)

// decoderBufferBytes is what encoding/base64 allocates for a value of chars
// characters: it sizes the output buffer from the character count alone and
// allocates it before validating anything, so this is the cost a rejected
// decode still pays.
func decoderBufferBytes(chars int) int {
	return chars / 4 * 3
}

// decodedLen counts a whole value handed over in one piece, which is the shape
// most cases below use.
func decodedLen(s string) int {
	var c xmlbase64.Counter
	c.Add([]byte(s))
	return c.DecodedLen()
}

// countPieces counts a value that arrives split, which is the shape a
// CipherValue spread over several text and CDATA nodes arrives in.
func countPieces(pieces []string) xmlbase64.Counter {
	var c xmlbase64.Counter
	for _, piece := range pieces {
		c.Add([]byte(piece))
	}
	return c
}

func TestDecodeString(t *testing.T) {
	t.Run("accepts XML whitespace between characters", func(t *testing.T) {
		decoded, err := xmlbase64.DecodeString(" QUJD\n\tREVG ")
		require.NoError(t, err)
		require.Equal(t, []byte("ABCDEF"), decoded)
	})

	t.Run("rejects non-whitespace junk", func(t *testing.T) {
		_, err := xmlbase64.DecodeString("!!!not-base64!!!")
		require.Error(t, err)
	})

	// A budget charges Counter.DecodedLen, which counts base64 characters and
	// ignores whitespace. If the decode sized its scratch buffer on the lexical
	// length instead, whitespace an attacker appends would allocate memory the budget
	// never charged, and the gap would grow with every byte of input. Checking
	// the decoded bytes alone cannot catch that, so this pins the allocation.
	//
	// The bound is checked via runtime.MemStats TotalAlloc, so this subtest
	// must NOT run in parallel (TotalAlloc is process-wide and a concurrent
	// test's allocations would pollute the delta).
	t.Run("allocates for base64 characters, not lexical length", func(t *testing.T) {
		// no t.Parallel(): isolated so the delta reflects only this decode.
		const lexical = 4 << 20
		// One quantum of payload; the rest is the whitespace
		// xs:base64Binary permits between characters.
		value := "AA==" + strings.Repeat(" \t\r\n", (lexical-4)/4)
		require.Greater(t, len(value), lexical-4)
		require.Equal(t, 1, decodedLen(value))

		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)
		decoded, err := xmlbase64.DecodeString(value)
		runtime.ReadMemStats(&after)
		runtime.KeepAlive(decoded)

		require.NoError(t, err)
		require.Equal(t, []byte{0}, decoded)

		// Sizing on the base64 characters costs a few dozen bytes here;
		// sizing on the lexical length costs about len(value). The bound is
		// far above the former and far below the latter, so it is immune to
		// allocator rounding and stray runtime allocations while still
		// failing outright on a lexical-length buffer.
		allocated := after.TotalAlloc - before.TotalAlloc
		require.Less(t, allocated, uint64(64<<10), "decoding %d lexical bytes allocated %d bytes", len(value), allocated)
	})
}

// TestDecodedLen covers the two halves of Counter.DecodedLen's contract on
// whole values: an exact count for input DecodeString accepts, and — for input
// it rejects — a count that never falls below the buffer the rejected decode
// allocates. The second half is what lets a caller weigh a value against a
// byte budget: a count that trusted a lexical form the decoder refuses would
// report less than the decode costs.
func TestDecodedLen(t *testing.T) {
	t.Run("exact for accepted input", func(t *testing.T) {
		for n := range 512 {
			data := make([]byte, n)
			for i := range data {
				data[i] = byte(i)
			}
			encoded := base64.StdEncoding.EncodeToString(data)
			require.Equal(t, n, decodedLen(encoded), "plain, n=%d", n)

			// The same value line-wrapped and indented, as real XML
			// Signature/Encryption producers emit it.
			wrapped := wrapWithWhitespace(encoded)
			decoded, err := xmlbase64.DecodeString(wrapped)
			require.NoError(t, err, "wrapped, n=%d", n)
			require.Len(t, decoded, n, "wrapped, n=%d", n)
			require.Equal(t, n, decodedLen(wrapped), "wrapped, n=%d", n)
		}
	})

	t.Run("whitespace-heavy accepted value counts exactly", func(t *testing.T) {
		// 3 bytes of payload buried in whitespace: the count must follow the
		// characters, not the string length.
		const s = "  \t\r\n Q U J \n D \t\r\n  "
		decoded, err := xmlbase64.DecodeString(s)
		require.NoError(t, err)
		require.Equal(t, []byte("ABC"), decoded)
		require.Equal(t, 3, decodedLen(s))
	})

	// Every value here is rejected by DecodeString, so there is no decoded
	// length to be exact about. What must hold is that the count covers the
	// decoder's buffer, which it allocates before rejecting.
	t.Run("rejected input is never under-counted", func(t *testing.T) {
		for _, tc := range []struct {
			name  string
			value string
			chars int
		}{
			// All padding. Deducting one byte per '=' would report 0.
			{name: "all padding", value: strings.Repeat("=", 4096), chars: 4096},
			// Padding mid-value. Every quantum is 2 of 4 padding, so
			// deducting per '=' would report half the decoder's buffer.
			{name: "repeated AA==", value: strings.Repeat("AA==", 1024), chars: 4096},
			// Padding ahead of the final quantum.
			{name: "padding before data", value: "==AAAAAA", chars: 8},
			// Three trailing '=' is one too many for any encoding.
			{name: "over-long padding", value: "A===", chars: 4},
			// Junk outside the alphabet: no padding to mis-deduct, but the
			// buffer is still allocated.
			{name: "outside the alphabet", value: strings.Repeat("!", 4096), chars: 4096},
			// Padding split by whitespace is still padding, and still not
			// at the end.
			{name: "whitespace between padding", value: "Q U = \n = B A = =", chars: 8},
			// Well-formed trailing padding on a body that is NOT base64.
			// The shape alone says two bytes may be deducted; the body
			// says the decoder will refuse the value and allocate the
			// whole quantum anyway.
			{name: "junk body with trailing padding", value: "!!==", chars: 4},
			// The same, with one alphabet character ahead of the junk.
			{name: "part-alphabet body with trailing padding", value: "A!==", chars: 4},
			// And the same value line-wrapped, the way an XML producer
			// would emit it.
			{name: "junk body with padding and whitespace", value: "! \t!\r\n==", chars: 4},
			// One '=' rather than two, so the deduction is smaller but
			// just as wrong.
			{name: "junk body with single trailing padding", value: "!!!=", chars: 4},
			// More than one quantum, so the shape check is not a
			// length-four special case.
			{name: "junk body with trailing padding, many quanta", value: strings.Repeat("!", 4094) + "==", chars: 4096},
		} {
			t.Run(tc.name, func(t *testing.T) {
				_, err := xmlbase64.DecodeString(tc.value)
				require.Error(t, err, "value must be rejected for this case to mean anything")
				require.GreaterOrEqual(t, decodedLen(tc.value), decoderBufferBytes(tc.chars))
			})
		}
	})

	t.Run("empty value counts zero", func(t *testing.T) {
		require.Equal(t, 0, decodedLen(""))
		require.Equal(t, 0, decodedLen(" \t\r\n"))
	})

	t.Run("never negative", func(t *testing.T) {
		for _, s := range []string{"=", "==", "===", "====", "A=", "==="} {
			require.GreaterOrEqual(t, decodedLen(s), 0, "value=%q", s)
		}
	})

	// Both halves of the contract at once, over every string the alphabet
	// below can build. The alphabet carries one character of each kind that
	// steers the count — two alphabet characters, padding, two whitespace
	// forms, and two characters outside the alphabet — so the enumeration
	// reaches every lexical shape the counter distinguishes rather than the
	// shapes anyone thought to name.
	t.Run("exhaustive over short values", func(t *testing.T) {
		const alphabet = "AB= \n!@"
		for _, s := range enumerate(alphabet, 6) {
			count := decodedLen(s)
			decoded, err := xmlbase64.DecodeString(s)
			if err != nil {
				require.GreaterOrEqual(t, count, decoderBufferBytes(strippedLen(s)), "rejected value=%q", s)
				continue
			}
			require.Equal(t, len(decoded), count, "accepted value=%q", s)
		}
	})
}

// TestCounter covers the property the split makes possible: a value fed in
// pieces counts exactly as its concatenation does. A caller that instead
// counted each piece alone and summed would not approximate that count, it
// would under-report it without limit — every piece shorter than a quantum
// counts 0 — and a budget charged that sum is no budget at all.
func TestCounter(t *testing.T) {
	// Each case is a value cut at a place where a piece-by-piece count differs
	// from the count of the whole: inside a quantum, inside the padding, and
	// around the junk that decides the shape.
	t.Run("split value counts as its concatenation", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			pieces []string
		}{
			// No piece is a whole quantum, so summing per piece reports 0.
			{name: "sub-quantum pieces", pieces: []string{"AAA", "AAA", "AAA", "AAA"}},
			// A thousand of the same, the shape the counter's doc names.
			{name: "many sub-quantum pieces", pieces: repeated("AAA", 1000)},
			// One quantum cut in four.
			{name: "quantum split character by character", pieces: []string{"A", "A", "B", "B"}},
			// Padding severed from the body it terminates: per piece the "=="
			// looks like a whole value of nothing but padding.
			{name: "padding split from body", pieces: []string{"QQ", "=="}},
			// The two padding characters split from each other.
			{name: "padding split in half", pieces: []string{"QQ=", "="}},
			// Whitespace-only pieces, as line-wrapped XML produces.
			{name: "whitespace-only pieces", pieces: []string{"QUJ", "\n\t  ", "D", "\n"}},
			// Junk in an earlier piece must still poison the padding
			// deduction claimed by a later one.
			{name: "junk ahead of well-formed padding", pieces: []string{"!!", "=="}},
			// Data after padding is only visible across the boundary.
			{name: "data after padding", pieces: []string{"QQ==", "QQQQ"}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				joined := strings.Join(tc.pieces, "")
				counter := countPieces(tc.pieces)
				require.Equal(t, decodedLen(joined), counter.DecodedLen())
				require.Equal(t, strippedLen(joined), counter.Chars())

				// And the count still covers the decode, whichever way the
				// decoder rules on the value.
				decoded, err := xmlbase64.DecodeString(joined)
				if err != nil {
					require.GreaterOrEqual(t, counter.DecodedLen(), decoderBufferBytes(strippedLen(joined)))
					return
				}
				require.Equal(t, len(decoded), counter.DecodedLen())
			})
		}
	})

	// The named cases above are the splits someone thought of. This one takes
	// values over an alphabet carrying every character kind that steers the
	// count and cuts each at random, so the carried state is exercised at
	// boundaries nobody chose.
	t.Run("randomized splits count as their concatenation", func(t *testing.T) {
		const alphabet = "AB=+/ \t\r\n!@"
		rng := rand.New(rand.NewSource(1))
		for range 20000 {
			value := randomValue(rng, alphabet, 24)
			pieces := randomSplit(rng, value)
			counter := countPieces(pieces)
			require.Equal(t, decodedLen(value), counter.DecodedLen(), "value=%q pieces=%q", value, pieces)
			require.Equal(t, strippedLen(value), counter.Chars(), "value=%q pieces=%q", value, pieces)

			// AppendStripped assembles the same pieces the counter sized, so
			// what a caller builds decodes to exactly what the whole value
			// does.
			var chars []byte
			for _, piece := range pieces {
				chars = xmlbase64.AppendStripped(chars, []byte(piece))
			}
			require.Len(t, chars, counter.Chars(), "value=%q pieces=%q", value, pieces)
			assembled, assembledErr := xmlbase64.DecodeString(string(chars))
			whole, wholeErr := xmlbase64.DecodeString(value)
			require.Equal(t, wholeErr == nil, assembledErr == nil, "value=%q pieces=%q", value, pieces)
			require.Equal(t, whole, assembled, "value=%q pieces=%q", value, pieces)
		}
	})

	t.Run("zero counter counts nothing", func(t *testing.T) {
		var c xmlbase64.Counter
		require.Equal(t, 0, c.DecodedLen())
		require.Equal(t, 0, c.Chars())
		c.Add(nil)
		c.Add([]byte(" \t\r\n"))
		require.Equal(t, 0, c.DecodedLen())
		require.Equal(t, 0, c.Chars())
	})
}

// repeated returns n copies of s as separate pieces.
func repeated(s string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = s
	}
	return out
}

// randomValue builds a value of up to maxLen characters drawn from alphabet.
func randomValue(rng *rand.Rand, alphabet string, maxLen int) string {
	var b strings.Builder
	for range rng.Intn(maxLen + 1) {
		b.WriteByte(alphabet[rng.Intn(len(alphabet))])
	}
	return b.String()
}

// randomSplit cuts s into pieces at random positions, empty pieces included:
// a DOM hands out whatever the document's node boundaries happen to be.
func randomSplit(rng *rand.Rand, s string) []string {
	cuts := []int{0, len(s)}
	for range rng.Intn(5) {
		cuts = append(cuts, rng.Intn(len(s)+1))
	}
	slices.Sort(cuts)
	pieces := make([]string, 0, len(cuts)-1)
	for i := 1; i < len(cuts); i++ {
		pieces = append(pieces, s[cuts[i-1]:cuts[i]])
	}
	return pieces
}

// enumerate returns every string of length 0 through maxLen over alphabet.
func enumerate(alphabet string, maxLen int) []string {
	out := []string{""}
	prev := []string{""}
	for range maxLen {
		next := make([]string, 0, len(prev)*len(alphabet))
		for _, s := range prev {
			for i := range len(alphabet) {
				next = append(next, s+string(alphabet[i]))
			}
		}
		out = append(out, next...)
		prev = next
	}
	return out
}

// strippedLen counts the characters left after DecodeString strips XML
// whitespace, which is what encoding/base64 sizes its output buffer from.
func strippedLen(s string) int {
	var n int
	for i := range len(s) {
		switch s[i] {
		case ' ', '\t', '\r', '\n':
		default:
			n++
		}
	}
	return n
}

// wrapWithWhitespace line-wraps s at 16 characters and indents each line,
// producing the interspersed XML whitespace xs:base64Binary permits.
func wrapWithWhitespace(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i += 16 {
		end := min(i+16, len(s))
		b.WriteString("\n\t  ")
		b.WriteString(s[i:end])
	}
	b.WriteString("\n  ")
	return b.String()
}

// parseValueElement parses markup as a document and returns its element, so a
// DecodeElement case gets a value laid out over exactly the text and CDATA
// children the markup describes.
func parseValueElement(t *testing.T, markup string) *helium.Element {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte("<v>"+markup+"</v>"))
	require.NoError(t, err)
	return doc.DocumentElement()
}

// parseEntityValueElement parses markup as the value element's content with
// decls as the document's internal DTD subset, so a reference in markup has an
// entity to name. helium's parser does not substitute entities, so the
// reference stays an EntityRefNode child of the element — the shape a
// conforming document that writes part of a base64 value as an entity
// reference actually hands DecodeElement.
func parseEntityValueElement(t *testing.T, decls, markup string) *helium.Element {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte("<!DOCTYPE v ["+decls+"]><v>"+markup+"</v>"))
	require.NoError(t, err)
	return doc.DocumentElement()
}

// parseWholeDocumentValueElement parses doc whole and returns its document
// element, for a case whose trigger lives in the prolog rather than in the
// value.
func parseWholeDocumentValueElement(t *testing.T, doc string) *helium.Element {
	t.Helper()
	parsed, err := helium.NewParser().Parse(t.Context(), []byte(doc))
	require.NoError(t, err)
	return parsed.DocumentElement()
}

// errCharge is what a caller's budget refusal looks like: DecodeElement must
// hand it back untouched so the caller can recognize its own error.
var errCharge = errors.New("charged too much")

// chargeAtMost returns a charge function that refuses anything over limit and
// records what it was asked for, so a case can assert the charge happened
// before any decode and for the right count.
func chargeAtMost(limit int, charged *int) func(int) error {
	return func(n int) error {
		*charged = n
		if n > limit {
			return errCharge
		}
		return nil
	}
}

func TestDecodeElement(t *testing.T) {
	t.Run("charges and decodes a value split across children", func(t *testing.T) {
		// The quantum boundaries fall inside children and inside CDATA
		// sections, so nothing here decodes correctly piece by piece.
		elem := parseValueElement(t, "QUJ<![CDATA[DR]]>\n\t EVG<![CDATA[R0hJ]]>")
		var charged int
		decoded, err := xmlbase64.DecodeElement(elem, chargeAtMost(1<<20, &charged))
		require.NoError(t, err)
		require.Equal(t, []byte("ABCDEFGHI"), decoded)
		require.Equal(t, 9, charged)
	})

	t.Run("charges nothing for an empty element", func(t *testing.T) {
		elem := parseValueElement(t, "")
		var charged int
		decoded, err := xmlbase64.DecodeElement(elem, chargeAtMost(1<<20, &charged))
		require.NoError(t, err)
		require.Empty(t, decoded)
		require.Equal(t, 0, charged)
	})

	t.Run("whitespace is not charged", func(t *testing.T) {
		// The value decodes to three bytes however much whitespace an attacker
		// wraps around it, so that is all the charge may be.
		elem := parseValueElement(t, "  QUJD<![CDATA[                    ]]>\n\n\n")
		var charged int
		decoded, err := xmlbase64.DecodeElement(elem, chargeAtMost(1<<20, &charged))
		require.NoError(t, err)
		require.Equal(t, []byte("ABC"), decoded)
		require.Equal(t, 3, charged)
	})

	t.Run("returns the charge error unchanged", func(t *testing.T) {
		elem := parseValueElement(t, "QUJDREVG")
		var charged int
		_, err := xmlbase64.DecodeElement(elem, chargeAtMost(2, &charged))
		require.ErrorIs(t, err, errCharge)
		require.Equal(t, 6, charged)
	})

	t.Run("reports the charge error for an over-budget invalid value", func(t *testing.T) {
		// The charge precedes the decode, so a value that is both over budget
		// and undecodable reports the budget rather than the base64 error. The
		// count is the decoder's own buffer size for those characters, which is
		// what a rejected decode would still allocate.
		elem := parseValueElement(t, "!!!!!!!!")
		var charged int
		_, err := xmlbase64.DecodeElement(elem, chargeAtMost(2, &charged))
		require.ErrorIs(t, err, errCharge)
		require.Equal(t, 6, charged)
	})

	t.Run("reports the base64 error for an in-budget invalid value", func(t *testing.T) {
		elem := parseValueElement(t, "QUJ<![CDATA[!]]>REVG")
		var charged int
		_, err := xmlbase64.DecodeElement(elem, chargeAtMost(1<<20, &charged))
		require.Error(t, err)
		require.NotErrorIs(t, err, errCharge)
		var corrupt base64.CorruptInputError
		require.ErrorAs(t, err, &corrupt)
	})

	// A comment contributes no character information item to its parent, so a
	// commented value must decode to exactly what the uncommented one does.
	// Reading the comment's text instead splices it into the base64 stream and
	// turns a legitimate value into a corrupt one.
	t.Run("comment and processing-instruction children contribute nothing", func(t *testing.T) {
		elem := parseValueElement(t, "QUJ<!--QUJD-->D<?pi QUJD?>REVG")
		var charged int
		decoded, err := xmlbase64.DecodeElement(elem, chargeAtMost(1<<20, &charged))
		require.NoError(t, err)
		require.Equal(t, []byte("ABCDEF"), decoded)
		require.Equal(t, 6, charged)
	})

	// xs:base64Binary is a simple type, so a ds:DigestValue/ds:SignatureValue
	// admits only character information items. An element child is therefore
	// invalid, and reading it would aggregate its whole subtree into one buffer
	// before anything is charged.
	t.Run("rejects an element child before charging", func(t *testing.T) {
		elem := parseValueElement(t, "QUJD<e>REVG</e>")
		charged := -1
		_, err := xmlbase64.DecodeElement(elem, chargeAtMost(1<<20, &charged))
		require.Error(t, err)
		require.NotErrorIs(t, err, errCharge)
		require.ErrorContains(t, err, "ElementNode")
		require.Equal(t, -1, charged, "the charge must not run once a child is refused")
	})

	// An entity reference stands for character data, so a document that writes
	// a value as one is conforming and must decode to what the same characters
	// written inline decode to.
	t.Run("reads an entity reference as its replacement text", func(t *testing.T) {
		elem := parseEntityValueElement(t, `<!ENTITY dv "QUJDREVG">`, "&dv;")
		var charged int
		decoded, err := xmlbase64.DecodeElement(elem, chargeAtMost(1<<20, &charged))
		require.NoError(t, err)
		require.Equal(t, []byte("ABCDEF"), decoded)
		require.Equal(t, 6, charged)
	})

	// The reference is one child among several and its characters do not fill a
	// quantum on their own, so this also pins that the counting state carries
	// across an entity-reference boundary the way it does across a CDATA one.
	t.Run("counts an entity reference across a child boundary", func(t *testing.T) {
		elem := parseEntityValueElement(t, `<!ENTITY dv "DR">`, "QUJ<![CDATA[]]>&dv;EVG")
		var charged int
		decoded, err := xmlbase64.DecodeElement(elem, chargeAtMost(1<<20, &charged))
		require.NoError(t, err)
		require.Equal(t, []byte("ABCDEF"), decoded)
		require.Equal(t, 6, charged)
	})

	// The Entity node hanging off an EntityRef is the declaration itself, and
	// its NextSibling is the NEXT declaration in the DTD's entity list rather
	// than more of this value. Only the first child may be read, so the two
	// unrelated declarations here must contribute nothing. Reading them would
	// splice "!!" into the stream and fail the decode.
	t.Run("reads only the referenced entity, not its declaration siblings", func(t *testing.T) {
		elem := parseEntityValueElement(t, `<!ENTITY a "QUJD"><!ENTITY b "!!"><!ENTITY c "!!">`, "&a;")
		var charged int
		decoded, err := xmlbase64.DecodeElement(elem, chargeAtMost(1<<20, &charged))
		require.NoError(t, err)
		require.Equal(t, []byte("ABC"), decoded)
		require.Equal(t, 3, charged)
	})

	// An entity's replacement text is taken raw: a nested reference inside it
	// stays the literal "&inner;" and is never expanded, so the value simply
	// does not decode. That is what keeps the read to one bounded step.
	t.Run("does not expand a nested entity reference", func(t *testing.T) {
		elem := parseEntityValueElement(t, `<!ENTITY inner "QUJD"><!ENTITY outer "&inner;">`, "&outer;")
		var charged int
		_, err := xmlbase64.DecodeElement(elem, chargeAtMost(1<<20, &charged))
		require.Error(t, err)
		require.NotErrorIs(t, err, errCharge)
		var corrupt base64.CorruptInputError
		require.ErrorAs(t, err, &corrupt)
	})

	// Markup in the replacement text is likewise raw characters, which are not
	// in the base64 alphabet, so the value fails the decode rather than being
	// read as a subtree.
	t.Run("does not read markup in an entity replacement as a subtree", func(t *testing.T) {
		elem := parseEntityValueElement(t, `<!ENTITY m "<x/>">`, "&m;")
		var charged int
		_, err := xmlbase64.DecodeElement(elem, chargeAtMost(1<<20, &charged))
		require.Error(t, err)
		require.NotErrorIs(t, err, errCharge)
		var corrupt base64.CorruptInputError
		require.ErrorAs(t, err, &corrupt)
	})

	// An undeclared general entity is a VALIDITY error, not a well-formedness
	// one, once the document has an external subset it did not read: a
	// non-validating processor keeps going and builds the reference with no
	// Entity linked to it. helium does exactly that, so a childless EntityRef is
	// ordinary parser output for a document any peer can send. It stands for no
	// character data — EntityRef.Content() is empty for it and c14n
	// canonicalizes it to nothing — so the value around it must decode as if the
	// reference were not there.
	t.Run("reads an undeclared entity reference under an external subset as no characters", func(t *testing.T) {
		elem := parseWholeDocumentValueElement(t, `<!DOCTYPE v SYSTEM "no-such.dtd"><v>QUJD&undeclared;REVG</v>`)
		var charged int
		decoded, err := xmlbase64.DecodeElement(elem, chargeAtMost(1<<20, &charged))
		require.NoError(t, err)
		require.Equal(t, []byte("ABCDEF"), decoded)
		require.Equal(t, 6, charged)
	})

	// An external subset is not the only way in. A parameter-entity reference in
	// the internal subset means the parser may not have seen every declaration
	// either, which lifts the same fatal check, so an entirely internal DTD
	// reaches the childless shape too.
	t.Run("reads an undeclared entity reference under a parameter-entity reference as no characters", func(t *testing.T) {
		elem := parseWholeDocumentValueElement(t, `<!DOCTYPE v [<!ENTITY % pe "<!ENTITY ignored 'x'>"> %pe;]><v>QUJD&undeclared;REVG</v>`)
		var charged int
		decoded, err := xmlbase64.DecodeElement(elem, chargeAtMost(1<<20, &charged))
		require.NoError(t, err)
		require.Equal(t, []byte("ABCDEF"), decoded)
		require.Equal(t, 6, charged)
	})

	// Contributing nothing is specific to the reference with no declaration. A
	// declared entity in the very same document still contributes its
	// replacement text, so the childless case cannot be read as entity
	// references having been dropped wholesale.
	t.Run("a declared entity still decodes beside an undeclared one", func(t *testing.T) {
		elem := parseWholeDocumentValueElement(t, `<!DOCTYPE v SYSTEM "no-such.dtd" [<!ENTITY dv "DR">]><v>QUJ&undeclared;&dv;EVG</v>`)
		var charged int
		decoded, err := xmlbase64.DecodeElement(elem, chargeAtMost(1<<20, &charged))
		require.NoError(t, err)
		require.Equal(t, []byte("ABCDEF"), decoded)
		require.Equal(t, 6, charged)
	})

	// The parser links a reference either to its declared Entity or to nothing
	// at all, so an EntityRef whose first child is some OTHER node is reachable
	// only from a caller-assembled DOM. Reading that child would be the
	// unbounded subtree read the first-child-plus-Entity shape exists to rule
	// out, so it is refused before the charge.
	t.Run("refuses an entity reference whose first child is not an entity", func(t *testing.T) {
		elem := parseValueElement(t, "QUJD")
		doc := elem.OwnerDocument()
		ref, err := doc.CreateReference("undeclared")
		require.NoError(t, err)
		payload, err := doc.CreateElement("payload")
		require.NoError(t, err)
		require.NoError(t, payload.AppendText([]byte("REVG")))
		require.NoError(t, ref.AddChild(payload))
		require.NoError(t, elem.AddChild(ref))

		charged := -1
		_, err = xmlbase64.DecodeElement(elem, chargeAtMost(1<<20, &charged))
		require.Error(t, err)
		require.NotErrorIs(t, err, errCharge)
		require.ErrorContains(t, err, "EntityRefNode")
		// The refusal names the shape it refuses — a first child that is not an
		// entity declaration — because a reference with NO child is accepted.
		require.ErrorContains(t, err, "whose first child is not an entity declaration")
		require.Equal(t, -1, charged, "the charge must not run once a child is refused")
	})

	// The decode buffer is sized from the same count the charge approved, so a
	// padded value never allocates the padding bytes encoding/base64 would size
	// from the character count alone. Sizing a decode buffer below what
	// encoding/base64 asks for is only safe if the count is exact for every
	// value it accepts, so this walks every payload length through all three
	// padding shapes rather than naming one.
	t.Run("sizes the decode buffer at the charged count", func(t *testing.T) {
		for n := range 256 {
			data := make([]byte, n)
			for i := range data {
				data[i] = byte(i)
			}
			elem := parseValueElement(t, wrapWithWhitespace(base64.StdEncoding.EncodeToString(data)))
			var charged int
			decoded, err := xmlbase64.DecodeElement(elem, chargeAtMost(1<<20, &charged))
			require.NoError(t, err, "n=%d", n)
			require.Equal(t, data, decoded, "n=%d", n)
			require.Equal(t, n, charged, "n=%d", n)
			require.Equal(t, charged, cap(decoded), "n=%d", n)
		}
	})

	t.Run("matches DecodeString on the joined children", func(t *testing.T) {
		// One text child, so the markup and the joined content are the same
		// string and DecodeString can be handed it directly.
		const value = "\n  QUJD\tREVG  R0hJSkxN\n"
		elem := parseValueElement(t, value)
		decoded, err := xmlbase64.DecodeElement(elem, chargeAtMost(1<<20, new(int)))
		require.NoError(t, err)
		want, err := xmlbase64.DecodeString(value)
		require.NoError(t, err)
		require.Equal(t, want, decoded)
	})
}

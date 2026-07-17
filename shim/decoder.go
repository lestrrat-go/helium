package shim

import (
	"bufio"
	"bytes"
	"context"
	stdxml "encoding/xml"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/sax"
)

// maxParseDepth is the maximum element nesting depth allowed by the shim.
// This guards against stack overflow from pathological input (CVE-2022-30633).
// Matches encoding/xml's maxUnmarshalDepth (10 000).
const maxParseDepth = 10_000

type tokenEvent struct {
	tok    Token
	rawTok Token // raw variant (prefix:local instead of namespace URI)
	line   int
	col    int
	err    error
	cdata  bool // true if this CharData came from a CDATA section
}

// Decoder reads XML tokens from a stream. It is a drop-in replacement for
// encoding/xml.Decoder backed by helium's SAX parser.
type Decoder struct {
	// Strict mode. When true (default), the parser requires strict XML conformance.
	Strict bool

	// AutoClose lists element names that should be auto-closed.
	AutoClose []string

	// Entity maps entity names to replacement text.
	Entity map[string]string

	// CharsetReader, if non-nil, defines a function to generate charset-conversion
	// readers, converting from the provided charset into UTF-8.
	CharsetReader func(charset string, input io.Reader) (io.Reader, error)

	// DefaultSpace sets the default namespace for elements without an explicit namespace.
	DefaultSpace string

	tokenReader     TokenReader
	events          chan tokenEvent
	done            <-chan struct{} // cancellation signal from cancel context; avoids storing context.Context
	ctxErr          func() error    // returns cancel context's error; avoids storing context.Context
	cancel          context.CancelFunc
	startSAX        func(io.Reader) // deferred SAX emitter start; captures cancel context in closure
	lastToken       Token
	savedErr        error
	offset          int64
	line            int
	column          int
	nestDepth       int         // tracks populateElement recursion depth
	prologTokens    []Token     // pre-scanned prolog tokens (Directive, ProcInst, Comment, CharData)
	prologIdx       int         // next prolog token to emit
	prologOnly      bool        // true if entire input is prolog (no root element)
	prologErr       error       // syntax error detected during prolog scanning
	combinedReader  io.Reader   // buffered reader for lazy SAX startup
	saxStarted      bool        // true once SAX goroutine has been started
	detectedCharset string      // non-UTF-8 encoding from XML declaration
	pendingEvent    *tokenEvent // lookahead event saved during CharData merging
	sawContent      bool        // TokenReader path: a non-whitespace token has been seen
}

func newDecoderFromReader(ctx context.Context, r io.Reader) (*Decoder, error) { //nolint:unparam // error always nil but callers check for future-proofing
	// Pre-scan the prolog to extract Directive, ProcInst, Comment, and
	// CharData tokens. The SAX parser does not emit these for the prolog,
	// so we handle them ourselves. The combined reader replays the full
	// input (including the prolog) for the SAX parser.
	prologTokens, combined, prologOnly, prologErr := scanProlog(r)

	ctx, cancel := context.WithCancel(ctx)
	d := &Decoder{
		Strict:         true,
		done:           ctx.Done(),
		ctxErr:         ctx.Err,
		cancel:         cancel,
		line:           1,
		column:         1,
		prologTokens:   prologTokens,
		prologOnly:     prologOnly || prologErr != nil,
		prologErr:      prologErr,
		combinedReader: combined,
	}
	// Capture the cancel context in a closure so the Decoder struct does not
	// store context.Context directly (satisfies containedctx linter).
	d.startSAX = func(r io.Reader) { d.startSAXEmitter(ctx, r) }
	// Error is always nil; kept in signature for compatibility.
	return d, nil
}

func newDecoderFromTokenReader(_ context.Context, tr TokenReader) *Decoder {
	return &Decoder{
		Strict:      true,
		tokenReader: tr,
		line:        1,
		column:      1,
	}
}

func (d *Decoder) startSAXEmitter(ctx context.Context, r io.Reader) {
	var locator sax.DocumentLocator

	push := func(tok, rawTok Token, line, col int) error {
		select {
		case d.events <- tokenEvent{tok: stdxml.CopyToken(tok), rawTok: stdxml.CopyToken(rawTok), line: line, col: col}:
			return nil
		case <-d.done:
			return d.ctxErr()
		}
	}

	// nsScope tracks in-scope namespace prefix→URI bindings across
	// nested elements so that prefixed attributes can be resolved even
	// when the namespace declaration is on an ancestor element.
	nsScope := map[string]string{}
	// nsScopeCounts tracks how many namespace declarations were pushed
	// per element level so they can be popped on EndElement.
	var nsScopeCounts []int
	// nsScopePrev tracks previous values for overridden prefixes so they
	// can be restored on EndElement.
	type nsPrev struct {
		prefix string
		uri    string
		had    bool
	}
	var nsScopePrevStack [][]nsPrev

	h := sax.New()
	h.SetOnStartDocument(sax.StartDocumentFunc(func(_ context.Context) error { return nil }))
	h.SetOnEndDocument(sax.EndDocumentFunc(func(_ context.Context) error { return nil }))
	h.SetOnSetDocumentLocator(sax.SetDocumentLocatorFunc(func(_ context.Context, loc2 sax.DocumentLocator) error {
		locator = loc2
		return nil
	}))
	pos := func() (line, col int) {
		if locator != nil {
			line = locator.LineNumber()
			col = locator.ColumnNumber()
		}
		return
	}
	h.SetOnStartElementNS(sax.StartElementNSFunc(func(_ context.Context, localname, prefix string, uri string, namespaces []sax.Namespace, attrs []sax.Attribute) error {
		line, col := pos()

		// Push newly declared namespaces into the in-scope map
		var prevs []nsPrev
		for _, ns := range namespaces {
			p := ns.Prefix()
			old, had := nsScope[p]
			prevs = append(prevs, nsPrev{prefix: p, uri: old, had: had})
			nsScope[p] = ns.URI()
		}
		nsScopeCounts = append(nsScopeCounts, len(namespaces))
		nsScopePrevStack = append(nsScopePrevStack, prevs)

		// Resolved token (for Token())
		se := StartElement{Name: Name{Space: uri, Local: localname}}
		// Raw token (for RawToken()) — prefix goes in Name.Space
		rawSE := StartElement{Name: Name{Space: prefix, Local: localname}}

		// Prepend namespace declarations as attributes (xmlns:* and xmlns)
		// before regular attributes, matching stdlib ordering.
		nAttr := len(namespaces) + len(attrs)
		if nAttr > 0 {
			se.Attr = make([]Attr, 0, nAttr)
			rawSE.Attr = make([]Attr, 0, nAttr)
			for _, ns := range namespaces {
				if ns.Prefix() == "" {
					a := Attr{Name: Name{Local: "xmlns"}, Value: ns.URI()}
					se.Attr = append(se.Attr, a)
					rawSE.Attr = append(rawSE.Attr, a)
				} else {
					a := Attr{Name: Name{Space: "xmlns", Local: ns.Prefix()}, Value: ns.URI()}
					se.Attr = append(se.Attr, a)
					rawSE.Attr = append(rawSE.Attr, a)
				}
			}
			for _, attr := range attrs {
				space := ""
				if p := attr.Prefix(); p != "" {
					space = nsScope[p]
				}
				se.Attr = append(se.Attr, Attr{
					Name:  Name{Space: space, Local: attr.LocalName()},
					Value: attr.Value(),
				})
				rawSE.Attr = append(rawSE.Attr, Attr{
					Name:  Name{Space: attr.Prefix(), Local: attr.LocalName()},
					Value: attr.Value(),
				})
			}
		}
		return push(se, rawSE, line, col)
	}))
	h.SetOnEndElementNS(sax.EndElementNSFunc(func(_ context.Context, localname, prefix string, uri string) error {
		line, col := pos()

		// Pop namespace declarations for this element
		if len(nsScopePrevStack) > 0 {
			prevs := nsScopePrevStack[len(nsScopePrevStack)-1]
			nsScopePrevStack = nsScopePrevStack[:len(nsScopePrevStack)-1]
			nsScopeCounts = nsScopeCounts[:len(nsScopeCounts)-1]
			for _, v := range slices.Backward(prevs) {
				p := v
				if p.had {
					nsScope[p.prefix] = p.uri
				} else {
					delete(nsScope, p.prefix)
				}
			}
		}

		ee := EndElement{Name: Name{Space: uri, Local: localname}}
		rawEE := EndElement{Name: Name{Space: prefix, Local: localname}}
		return push(ee, rawEE, line, col)
	}))
	h.SetOnCharacters(sax.CharactersFunc(func(_ context.Context, ch []byte) error {
		line, col := pos()
		cd := CharData(append([]byte(nil), ch...))
		return push(cd, cd, line, col)
	}))
	h.SetOnIgnorableWhitespace(sax.IgnorableWhitespaceFunc(func(_ context.Context, ch []byte) error {
		line, col := pos()
		cd := CharData(append([]byte(nil), ch...))
		return push(cd, cd, line, col)
	}))
	h.SetOnCDataBlock(sax.CDataBlockFunc(func(_ context.Context, value []byte) error {
		line, col := pos()
		cd := CharData(append([]byte(nil), value...))
		select {
		case d.events <- tokenEvent{tok: cd, rawTok: cd, line: line, col: col, cdata: true}:
			return nil
		case <-d.done:
			return d.ctxErr()
		}
	}))
	h.SetOnComment(sax.CommentFunc(func(_ context.Context, value []byte) error {
		line, col := pos()
		c := Comment(append([]byte(nil), value...))
		return push(c, c, line, col)
	}))
	h.SetOnProcessingInstruction(sax.ProcessingInstructionFunc(func(_ context.Context, target, data string) error {
		// Suppress only the lowercase "xml" declaration PI: helium re-parses the
		// blanked-declaration document, so this callback only ever fires for PIs
		// helium accepts, and helium rejects a reserved-cased target upstream —
		// none reaches here. An exact match is deliberate: folding here would
		// silently DROP such a target rather than reject it, if one ever did.
		if target == lexicon.PrefixXML {
			return nil // skip XML declaration
		}
		line, col := pos()
		pi := ProcInst{Target: target, Inst: []byte(data)}
		return push(pi, pi, line, col)
	}))

	// Stubs for callbacks we don't use
	h.SetOnInternalSubset(sax.InternalSubsetFunc(func(_ context.Context, _ string, _ string, _ string) error { return nil }))
	h.SetOnExternalSubset(sax.ExternalSubsetFunc(func(_ context.Context, _ string, _ string, _ string) error { return nil }))
	h.SetOnReference(sax.ReferenceFunc(func(_ context.Context, _ string) error { return nil }))
	h.SetOnEntityDecl(sax.EntityDeclFunc(func(_ context.Context, _ string, _ enum.EntityType, _ string, _ string, _ string) error { return nil }))
	h.SetOnElementDecl(sax.ElementDeclFunc(func(_ context.Context, _ string, _ enum.ElementType, _ sax.ElementContent) error { return nil }))
	h.SetOnAttributeDecl(sax.AttributeDeclFunc(func(_ context.Context, _ string, _ string, _ enum.AttributeType, _ enum.AttributeDefault, _ string, _ sax.Enumeration) error {
		return nil
	}))
	h.SetOnNotationDecl(sax.NotationDeclFunc(func(_ context.Context, _ string, _ string, _ string) error { return nil }))
	h.SetOnUnparsedEntityDecl(sax.UnparsedEntityDeclFunc(func(_ context.Context, _ string, _ string, _ string, _ string) error { return nil }))
	// Pre-build helium entities from d.Entity so the parser can type-assert
	// them to *helium.Entity (the parser hard-casts in parseEntityRef).
	var entityLookup map[string]*helium.Entity
	if len(d.Entity) > 0 {
		entDoc := helium.NewDefaultDocument()
		if _, err := entDoc.CreateInternalSubset("_", "", ""); err == nil {
			entityLookup = make(map[string]*helium.Entity, len(d.Entity))
			for name, val := range d.Entity {
				ent, err := entDoc.AddEntity(name, enum.InternalGeneralEntity, "", "", val)
				if err == nil {
					entityLookup[name] = ent
				}
			}
		}
	}
	h.SetOnGetEntity(sax.GetEntityFunc(func(_ context.Context, name string) (sax.Entity, error) {
		if entityLookup != nil {
			if ent, ok := entityLookup[name]; ok {
				return ent, nil
			}
		}
		return nil, nil //nolint:nilnil
	}))
	h.SetOnGetParameterEntity(sax.GetParameterEntityFunc(func(_ context.Context, _ string) (sax.Entity, error) { return nil, nil }))     //nolint:nilnil
	h.SetOnResolveEntity(sax.ResolveEntityFunc(func(_ context.Context, _ string, _ string) (sax.ParseInput, error) { return nil, nil })) //nolint:nilnil
	h.SetOnHasExternalSubset(sax.HasExternalSubsetFunc(func(_ context.Context) (bool, error) { return false, nil }))
	h.SetOnHasInternalSubset(sax.HasInternalSubsetFunc(func(_ context.Context) (bool, error) { return false, nil }))
	h.SetOnIsStandalone(sax.IsStandaloneFunc(func(_ context.Context) (bool, error) { return false, nil }))
	h.SetOnError(sax.ErrorFunc(func(_ context.Context, err error) error { return err }))
	h.SetOnWarning(sax.WarningFunc(func(_ context.Context, _ error) error { return nil }))

	go func() {
		defer close(d.events)
		// The parser is handed a document whose XML declaration has been
		// blanked out to spaces by scanProlog, which tokenized the prolog
		// itself. So the parser never parses a declaration in context on this
		// path; checkXMLDecl instead reconstructs the drained declaration and asks
		// helium to judge it, so the verdict is still helium's.
		p := helium.NewParser().MaxDepth(maxParseDepth).SAXHandler(h)
		_, err := p.ParseReader(ctx, r)
		if err != nil {
			select {
			case d.events <- tokenEvent{err: err}:
			case <-d.done:
			}
		}
	}()
}

// Close cancels the SAX goroutine and releases resources.
func (d *Decoder) Close() {
	if d.cancel != nil {
		d.cancel()
	}
}

func (d *Decoder) advancePosition(tok Token) {
	// Estimate byte size from token for InputOffset tracking.
	n := tokenSize(tok)
	d.offset += int64(n)
}

// tokenSize returns an estimated byte size of the serialized token,
// matching encoding/xml's offset accounting.
func tokenSize(tok Token) int {
	switch v := tok.(type) {
	case StartElement:
		// <name attr="val">
		n := 1 + len(v.Name.Local) + 1 // < name >
		// v.Name.Space is not used here (approximation; we don't have the prefix)
		for _, a := range v.Attr {
			n += 1 + len(a.Name.Local) + 2 + len(a.Value) + 1 // space name="val"
		}
		return n
	case EndElement:
		return 2 + len(v.Name.Local) + 1 // </name>
	case CharData:
		return len(v)
	case Comment:
		return 7 + len(v) // <!--...-->
	case ProcInst:
		return 4 + len(v.Target) + 1 + len(v.Inst) + 2 // <?target data?>
	case Directive:
		return 3 + len(v) + 1 // <!...>
	}
	return 0
}

// Token returns the next XML token in the input stream.
// Namespace URIs are resolved in the Name.Space field.
func (d *Decoder) Token() (Token, error) {
	tok, err := d.readToken(false)
	if err != nil {
		return nil, err
	}
	if d.DefaultSpace != "" {
		tok = applyDefaultSpace(tok, d.DefaultSpace)
	}
	return tok, nil
}

// RawToken returns the next XML token without namespace resolution.
// Element names use prefix:local form instead of resolved namespace URIs.
func (d *Decoder) RawToken() (Token, error) {
	return d.readToken(true)
}

func (d *Decoder) readToken(raw bool) (Token, error) {
	// Drain pre-scanned prolog tokens first.
	if d.prologIdx < len(d.prologTokens) {
		tok := stdxml.CopyToken(d.prologTokens[d.prologIdx])
		d.prologIdx++
		d.lastToken = tok
		d.advancePosition(tok)

		// A ProcInst whose target is the reserved "xml" name (in any casing) is
		// held to the declaration gate: the lowercase "xml" is the XML
		// declaration, which checkXMLDecl reconstructs and lets helium judge
		// (grammar and version), and any other casing is an illegal PITarget that
		// checkXMLDecl rejects.
		if pi, ok := tok.(ProcInst); ok && isReservedXMLTarget(pi.Target) {
			if err := d.checkXMLDecl(pi.Target, string(pi.Inst)); err != nil {
				return nil, err
			}
		}

		return tok, nil
	}

	// If the prolog scanner detected a syntax error, return it after
	// draining any prolog tokens that were successfully parsed.
	if d.prologErr != nil {
		err := d.prologErr
		d.prologErr = nil
		return nil, err
	}

	// Lazy-start the SAX emitter on first non-prolog read. This allows
	// callers to set Entity, CharsetReader, etc. after NewDecoder returns.
	if !d.saxStarted && d.tokenReader == nil {
		d.saxStarted = true
		if d.prologOnly {
			d.events = make(chan tokenEvent)
			close(d.events)
		} else {
			reader := d.combinedReader
			// A fixed-width Unicode declaration is invisible to scanProlog, so its
			// encoding gate never ran; apply the policy here from helium's decoded
			// encoding before the streaming parse.
			fixedReader, err := d.applyFixedWidthEncodingPolicy(reader)
			if err != nil {
				d.combinedReader = nil
				return nil, err
			}
			reader = fixedReader
			if d.detectedCharset != "" && d.CharsetReader != nil {
				newr, err := d.CharsetReader(d.detectedCharset, bufio.NewReader(reader))
				if err != nil {
					d.combinedReader = nil
					return nil, fmt.Errorf("xml: opening charset %q: %w", d.detectedCharset, err)
				}
				if newr == nil {
					d.combinedReader = nil
					return nil, fmt.Errorf("xml: CharsetReader returned nil Reader for charset %q", d.detectedCharset)
				}
				reader = ensureReader(newr)
			}
			d.events = make(chan tokenEvent, 64)
			d.startSAX(reader)
		}
		d.combinedReader = nil // release reference
	}

	var tok Token

	if d.tokenReader != nil {
		if d.savedErr != nil {
			err := d.savedErr
			d.savedErr = nil
			return nil, err
		}
		nextTok, err := d.tokenReader.Token()
		// Pass (nil, nil) straight through to match encoding/xml exactly.
		// Per the TokenReader contract, (nil, nil) means "nothing happened"
		// (e.g. a polling/non-blocking reader with no token available yet) and
		// is NOT EOF; stdlib's Decoder.Token() returns it verbatim to the
		// caller rather than retrying or erroring. Diverging here would break
		// drop-in stdlib compatibility. The shim's own internal driving loops
		// (Decode/populateElement, Skip) carry a bounded no-progress guard so
		// the shim itself can never hang on a pathological reader.
		if nextTok == nil && err == nil {
			return nil, nil //nolint:nilnil // stdlib parity: (nil, nil) means "nothing happened"
		}
		if err != nil && nextTok == nil {
			return nil, err
		}
		// Token came back with a trailing error: return the token now and
		// surface the error on the next read (the general TokenReader case).
		if err != nil {
			d.savedErr = err
		}
		tok = nextTok
	} else {
		event, ok := d.nextEvent()
		if !ok {
			return nil, io.EOF
		}
		if event.err != nil {
			return nil, convertParseError(event.err)
		}
		if event.line > 0 {
			d.line = event.line
			d.column = event.col
		}

		// Merge consecutive CharData events into a single token.
		// The SAX parser fires separate callbacks for each entity/character
		// reference expansion, but stdlib returns one merged CharData.
		if _, isCD := event.tok.(CharData); isCD {
			event = d.mergeCharData(event, raw)
		}

		if raw {
			tok = event.rawTok
		} else {
			tok = event.tok
		}
	}

	tok = stdxml.CopyToken(tok)

	// A ProcInst whose target is the reserved "xml" name (in any casing) is held
	// to the same rules wherever it came from, including a TokenReader. The
	// lowercase "xml" is the XML declaration; any other casing is an illegal
	// PITarget. The SAX emitter suppresses its own declaration, so one only ever
	// reaches this point from a TokenReader.
	if pi, ok := tok.(ProcInst); ok && isReservedXMLTarget(pi.Target) {
		// XMLDecl is only legal as the very first thing in a document
		// (prolog ::= XMLDecl? Misc* ...), with only whitespace ahead of it.
		// d.sawContent records whether a non-whitespace token already reached
		// the caller on the TokenReader path — mirroring prologScanner.sawContent
		// on the reader path — so a declaration after a comment, PI, doctype,
		// earlier declaration, or the root start tag is rejected here too.
		if d.sawContent {
			return nil, errDeclNotAtStart
		}
		if err := d.checkXMLDecl(pi.Target, string(pi.Inst)); err != nil {
			return nil, err
		}
	}

	// Record prior content for the TokenReader path's placement rule above.
	// Leading whitespace and a leading byte-order mark do not count (isLeadingNoise),
	// matching the reader path, where scanProlog accumulates whitespace as CharData
	// without setting sawContent and a BOM is consumed by helium ahead of the
	// declaration.
	if d.tokenReader != nil && !isLeadingNoise(tok) {
		d.sawContent = true
	}

	d.lastToken = tok
	d.advancePosition(tok)
	return tok, nil
}

// isReservedXMLTarget reports whether target is the reserved processing-
// instruction target "xml" in ANY casing. XML 1.0 §2.6 reserves it via
//
//	PITarget ::= Name - (('X'|'x')('M'|'m')('L'|'l'))
//
// so "xml", "XML", "Xml", "xMl", … are all reserved: only the lowercase "xml"
// introduces an XML declaration, and every other casing is an illegal target.
// A longer xml-prefixed name such as "xmlfoo" or "xml-stylesheet" is an
// ordinary, legal target; strings.EqualFold matches only equal-length strings,
// so those longer names are left untouched. This is the single classifier the
// declaration gate uses on all three decode paths, so the reserved-target rule
// cannot drift apart from the grammar/version/encoding rules.
func isReservedXMLTarget(target string) bool {
	return strings.EqualFold(target, lexicon.PrefixXML)
}

// checkXMLDecl is the declaration gate for a ProcInst whose target is the
// reserved "xml" name, used on the Decoder paths where helium does not otherwise
// judge the declaration in context (the reader path blanks it out of the bytes
// replayed to the parser; the TokenReader path carries no bytes at all). Only
// the lowercase "xml" introduces an XML declaration; a target in any other
// casing is an illegal PITarget (isReservedXMLTarget) and is rejected before its
// pseudo-attributes are read. For the lowercase target the declaration is
// reconstructed and handed to helium — the single authority for the XMLDecl
// grammar and the version rule — and the encoding it declares is then held to
// the shim's CharsetReader rule.
func (d *Decoder) checkXMLDecl(target, data string) error {
	if target != lexicon.PrefixXML {
		return &stdxml.SyntaxError{
			Msg:  fmt.Sprintf("%q is a reserved processing-instruction target", target),
			Line: d.line,
		}
	}
	enc, err := heliumDeclDecision(data)
	if err != nil {
		return err
	}
	return d.checkDeclEncoding(enc)
}

// declProbeRoot is the minimal well-formed root appended to a reconstructed
// declaration so helium has a complete document to parse.
const declProbeRoot = "<r/>"

// heliumDeclDecision is the shim's single declaration-decision function. It
// reconstructs the declaration as a standalone document — "<?xml " + data + "?>"
// followed by a minimal root — and hands it to helium, whose parse is the
// authority for the XMLDecl grammar (XML 1.0 §2.8) and the version rule: 1.0 and
// 1.1 are supported (helium implements XML 1.1), and a version outside the 1.x
// family is rejected. Every Decoder path that has a declaration but no in-context
// helium parse routes its verdict through here, so the grammar and version rules
// cannot drift from helium's. On success it returns the declared encoding (the
// "utf8" sentinel when none was declared) for the CharsetReader rule; on failure
// it returns helium's verdict as an [encoding/xml.SyntaxError].
func heliumDeclDecision(data string) (string, error) {
	src := "<?xml " + data + "?>" + declProbeRoot
	p := helium.NewParser().MaxDepth(maxParseDepth)
	doc, err := p.Parse(context.Background(), []byte(src))
	if err != nil {
		return "", convertParseError(err)
	}
	return doc.Encoding(), nil
}

// checkDeclEncoding applies the shim's encoding rule to a declaration's
// EncName. UTF-8 (case-insensitive) is always accepted, as is helium's
// no-encoding sentinel. Any other encoding requires a CharsetReader to convert
// it.
func (d *Decoder) checkDeclEncoding(enc string) error {
	if !encodingNeedsCharsetReader(enc) {
		return nil
	}
	if d.CharsetReader == nil {
		return errCharsetReaderNil(enc)
	}
	d.detectedCharset = enc
	return nil
}

// encodingNeedsCharsetReader reports whether enc is an explicitly declared
// non-UTF-8 encoding that a CharsetReader is required to honor. The empty string
// and helium's no-encoding sentinel both mean no encoding was declared, and UTF-8
// (case-insensitive) needs no conversion. This is the shim's single encoding
// classifier, shared by every entry point so their verdicts cannot drift.
func encodingNeedsCharsetReader(enc string) bool {
	if enc == "" || enc == heliumNoEncoding {
		return false
	}
	return !strings.EqualFold(enc, "utf-8")
}

// errCharsetReaderNil is the error every entry point returns when a document
// declares a non-UTF-8 encoding that cannot be honored because no CharsetReader
// is configured. Building it in one place keeps the message identical wherever
// the policy fires.
func errCharsetReaderNil(enc string) error {
	return fmt.Errorf("xml: encoding %q declared but Decoder.CharsetReader is nil", enc)
}

// applyFixedWidthEncodingPolicy applies the shim's encoding policy on the
// reader-backed Decoder path when the declaration is written in a fixed-width
// Unicode encoding (UTF-16 / UCS-4). The byte-level prolog scanner cannot
// tokenize such a declaration, so its encoding gate (checkXMLDecl) never fires
// here. When the stream begins with a fixed-width Unicode marker, the whole
// document is decoded by helium — the authority on the declared encoding — and a
// declared non-UTF-8 encoding without a CharsetReader is rejected, matching
// Unmarshal and the scanner's own gate. The returned reader replays the same
// bytes for the streaming parse; ASCII-compatible streams are returned untouched
// so the fast path is undisturbed.
func (d *Decoder) applyFixedWidthEncodingPolicy(r io.Reader) (io.Reader, error) {
	br := bufio.NewReader(r)
	prefix, _ := br.Peek(4)
	if !looksFixedWidthUnicode(prefix) {
		return br, nil
	}
	buf, err := io.ReadAll(br)
	if err != nil {
		return nil, err
	}
	doc, err := helium.NewParser().MaxDepth(maxParseDepth).Parse(context.Background(), buf)
	if err != nil {
		return nil, convertParseError(err)
	}
	enc := doc.Encoding()
	if encodingNeedsCharsetReader(enc) && d.CharsetReader == nil {
		return nil, errCharsetReaderNil(enc)
	}
	return bytes.NewReader(buf), nil
}

// looksFixedWidthUnicode reports whether prefix begins a fixed-width Unicode
// stream (UTF-16 / UCS-4) that the byte-level prolog scanner cannot read: a
// UTF-16 byte-order mark, or a NUL among the leading bytes (present in every
// UCS-4 or UTF-16 encoding of the leading ASCII '<'). A UTF-8 or ASCII document,
// with or without a UTF-8 byte-order mark, has neither, so it stays on the fast
// path. A false positive only costs a decode that helium then judges correctly.
func looksFixedWidthUnicode(prefix []byte) bool {
	if len(prefix) >= 2 {
		if prefix[0] == 0xFF && prefix[1] == 0xFE {
			return true
		}
		if prefix[0] == 0xFE && prefix[1] == 0xFF {
			return true
		}
	}
	return bytes.IndexByte(prefix, 0x00) >= 0
}

func applyDefaultSpace(tok Token, space string) Token {
	switch v := tok.(type) {
	case StartElement:
		if v.Name.Space == "" {
			v.Name.Space = space
			return v
		}
	case EndElement:
		if v.Name.Space == "" {
			v.Name.Space = space
			return v
		}
	}
	return tok
}

// nextEvent returns the next event, consuming the pending lookahead if available.
func (d *Decoder) nextEvent() (tokenEvent, bool) {
	if d.pendingEvent != nil {
		ev := *d.pendingEvent
		d.pendingEvent = nil
		return ev, true
	}
	ev, ok := <-d.events
	return ev, ok
}

// mergeCharData coalesces consecutive non-CDATA CharData events into one.
// CDATA sections are kept as separate tokens to match stdlib behavior.
// It reads ahead from the event channel until a non-mergeable event is found
// (which is saved as pendingEvent for the next call).
func (d *Decoder) mergeCharData(first tokenEvent, _ bool) tokenEvent {
	if first.cdata {
		return first
	}
	merged := first
	cookedBuf := []byte(merged.tok.(CharData)) //nolint:forcetypeassert
	rawBuf := []byte(merged.rawTok.(CharData)) //nolint:forcetypeassert
	for {
		next, ok := <-d.events
		if !ok {
			break
		}
		if next.err != nil {
			d.pendingEvent = &next
			break
		}
		nextCD, isCD := next.tok.(CharData)
		if !isCD || next.cdata {
			d.pendingEvent = &next
			break
		}
		cookedBuf = append(cookedBuf, nextCD...)
		rawBuf = append(rawBuf, next.rawTok.(CharData)...) //nolint:forcetypeassert
	}
	merged.tok = CharData(cookedBuf)
	merged.rawTok = CharData(rawBuf)
	return merged
}

// maxNoProgress bounds how many consecutive (nil, nil) "nothing happened"
// reads the shim's own internal driving loops tolerate before giving up.
// Token() passes (nil, nil) straight through to the caller for stdlib parity,
// but loops that drive the decoder internally (Decode/populateElement, Skip)
// expect forward progress; without this bound a pathological TokenReader that
// always returns (nil, nil) would hang the shim itself.
const maxNoProgress = 10_000

// driveToken reads the next token for an internal driving loop. A (nil, nil)
// "nothing happened" result is retried up to maxNoProgress times before the
// loop is failed with io.ErrNoProgress, so the shim can never hang internally.
func (d *Decoder) driveToken() (Token, error) {
	for range maxNoProgress {
		tok, err := d.Token()
		if err != nil {
			return nil, err
		}
		if tok != nil {
			return tok, nil
		}
	}
	return nil, io.ErrNoProgress
}

func (d *Decoder) Skip() error {
	if d.lastToken == nil {
		return errors.New("shim: Skip called before reading start element")
	}
	if _, ok := d.lastToken.(StartElement); !ok {
		return nil
	}

	depth := 1
	for depth > 0 {
		tok, err := d.driveToken()
		if err != nil {
			return err
		}
		switch tok.(type) {
		case StartElement:
			depth++
		case EndElement:
			depth--
		}
	}
	return nil
}

func (d *Decoder) InputOffset() int64 {
	return d.offset
}

func (d *Decoder) InputPos() (line, column int) {
	if d.line == 0 {
		return 1, 1
	}
	return d.line, d.column
}

func (d *Decoder) Decode(v any) error {
	if _, err := validateUnmarshalTarget(v); err != nil {
		return err
	}
	for {
		tok, err := d.driveToken()
		if err != nil {
			return err
		}
		start, ok := tok.(stdxml.StartElement)
		if !ok {
			continue
		}
		return d.DecodeElement(v, &start)
	}
}

func (d *Decoder) DecodeElement(v any, start *StartElement) error {
	rv, err := validateUnmarshalTarget(v)
	if err != nil {
		return err
	}
	if start == nil {
		return d.Decode(v)
	}

	elem, err := d.buildElementFromTokens(*start)
	if err != nil {
		return err
	}
	return decodeElementInto(rv, elem)
}

// buildElementFromTokens reads tokens from the decoder and builds
// a helium Element subtree. This avoids the previous approach of
// serializing tokens to bytes and re-parsing.
func (d *Decoder) buildElementFromTokens(start stdxml.StartElement) (*helium.Element, error) {
	doc := helium.NewDefaultDocument()

	root := doc.CreateElement(start.Name.Local)

	// Set namespace if present
	if start.Name.Space != "" {
		if err := root.SetActiveNamespace("", start.Name.Space); err != nil {
			return nil, err
		}
	}

	// Set attributes
	if err := setElementAttrs(doc, root, start.Attr); err != nil {
		return nil, err
	}

	if err := doc.SetDocumentElement(root); err != nil {
		return nil, err
	}

	// Read children
	if err := d.populateElement(doc, root, start.Name); err != nil {
		return nil, err
	}

	return root, nil
}

// maxNestingDepth limits how deeply populateElement can recurse,
// protecting against malicious TokenReaders that never return EndElement.
const maxNestingDepth = 10000

func (d *Decoder) populateElement(doc *helium.Document, parent *helium.Element, name Name) error {
	d.nestDepth++
	if d.nestDepth > maxNestingDepth {
		return errors.New("xml: exceeded max depth")
	}
	defer func() { d.nestDepth-- }()
	for {
		tok, err := d.driveToken()
		if err != nil {
			if err == io.EOF {
				return io.ErrUnexpectedEOF
			}
			return err
		}

		switch v := tok.(type) {
		case StartElement:
			child := doc.CreateElement(v.Name.Local)
			if v.Name.Space != "" {
				if err := child.SetActiveNamespace("", v.Name.Space); err != nil {
					return err
				}
			}
			if err := setElementAttrs(doc, child, v.Attr); err != nil {
				return err
			}
			if err := parent.AddChild(child); err != nil {
				return err
			}
			if err := d.populateElement(doc, child, v.Name); err != nil {
				return err
			}
		case EndElement:
			if v.Name.Local != name.Local || v.Name.Space != name.Space {
				return &SyntaxError{
					Msg:  "element <" + name.Local + "> closed by </" + v.Name.Local + ">",
					Line: d.line,
				}
			}
			return nil
		case CharData:
			text := doc.CreateText([]byte(v))
			if err := parent.AddChild(text); err != nil {
				return err
			}
		case Comment:
			comment := doc.CreateComment([]byte(v))
			if err := parent.AddChild(comment); err != nil {
				return err
			}
		case ProcInst:
			pi := doc.CreatePI(v.Target, string(v.Inst))
			if err := parent.AddChild(pi); err != nil {
				return err
			}
		}
	}
}

// setElementAttrs sets attributes on an element, preserving namespace URIs
// so that lookupAttr can match namespace-qualified attribute bindings.
// ensureReader wraps r so it works with io.Read even if r only
// implements io.ByteReader (e.g., CharsetReader returns that pattern).
func ensureReader(r io.Reader) io.Reader {
	if br, ok := r.(io.ByteReader); ok {
		return &byteReaderWrapper{br: br}
	}
	return r
}

// byteReaderWrapper adapts an io.ByteReader into a full io.Reader.
type byteReaderWrapper struct {
	br io.ByteReader
}

func (w *byteReaderWrapper) Read(p []byte) (int, error) {
	for i := range p {
		b, err := w.br.ReadByte()
		if err != nil {
			return i, err
		}
		p[i] = b
	}
	return len(p), nil
}

func setElementAttrs(doc *helium.Document, elem *helium.Element, attrs []stdxml.Attr) error {
	for _, attr := range attrs {
		if attr.Name.Space != "" {
			ns, err := doc.CreateNamespace("_", attr.Name.Space)
			if err != nil {
				return err
			}
			if _, err := elem.SetAttributeNS(attr.Name.Local, attr.Value, ns); err != nil {
				return err
			}
		} else {
			if _, err := elem.SetAttribute(attr.Name.Local, attr.Value); err != nil {
				return err
			}
		}
	}
	return nil
}

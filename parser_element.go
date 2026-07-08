package helium

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/strcursor"
	"github.com/lestrrat-go/helium/internal/xmlchar"
	"github.com/lestrrat-go/helium/sax"
)

/* parse a CharData section.
 * if we are within a CDATA section ']]>' marks an end of section.
 *
 * The right angle bracket (>) may be represented using the string "&gt;",
 * and must, for compatibility, be escaped using "&gt;" or a character
 * reference when it appears in the string "]]>" in content, when that
 * string is not marking the end of a CDATA section.
 *
 * [14] CharData ::= [^<&]* - ([^<&]* ']]>' [^<&]*)
 */
func (pctx *parserCtx) parseCharData(ctx context.Context, cdata bool) error {
	if cdata {
		_, err := pctx.parseCDataContent()
		return err
	}
	return pctx.parseCharDataContent(ctx)
}

func (pctx *parserCtx) parseCharDataContent(ctx context.Context) error {
	cur := pctx.getCursor()
	if cur == nil {
		return pctx.error(ctx, errNoCursor)
	}

	// Fast path: UTF8Cursor can scan directly into a []byte slice,
	// avoiding the bytes.Buffer intermediate.
	if u8, ok := cur.(*strcursor.UTF8Cursor); ok {
		// Streaming SAX consumers that configured a char-buffer size get
		// bounded memory: scan and deliver the run in fixed-size chunks rather
		// than buffering the whole delimiter-free run (which would also grow the
		// cursor's internal buffer) before chunking only the delivery.
		// pctx.doc == nil ensures no DOM is being built. A SAX wrapper that
		// delegates to a TreeBuilder has pctx.treeBuilder == nil (it is not the
		// concrete *TreeBuilder) yet pctx.doc is populated (TreeBuilder.StartDocument
		// set it). Such wrappers must use the single-shot classification path so a
		// large whitespace run is classified over the whole run and delivered via
		// IgnorableWhitespace (which StripBlanks drops) rather than being downgraded
		// to Characters by the chunked path's blankBudget cap.
		if pctx.charBufferSize > 0 && pctx.treeBuilder == nil && pctx.doc == nil &&
			pctx.sax != nil && !pctx.disableSAX {
			return pctx.parseCharDataChunkedSAX(ctx, u8)
		}

		// Bound the scan to the node-content cap (plus a rune of slack) so an
		// oversized delimiter-free run is detected and rejected before the whole
		// run — and the cursor's internal buffer — is materialized.
		data, i := u8.ScanCharDataSlice(pctx.charBuf[:0], pctx.nodeContentScanBudget())
		if i <= 0 {
			if cur.Peek() == ']' && cur.PeekAt(1) == ']' && cur.PeekAt(2) == '>' {
				return pctx.error(ctx, ErrMisplacedCDATAEnd)
			}
			return errors.New("invalid char data")
		}
		if pctx.nodeContentTooLong(i) {
			return pctx.error(ctx, ErrNodeContentTooLarge)
		}

		if err := cur.AdvanceFast(i); err != nil {
			return err
		}

		// Keep the grown buffer for next call.
		pctx.charBuf = data

		if pctx.areBlanksBytes(data, false) {
			if pctx.treeBuilder != nil && !pctx.disableSAX {
				if err := pctx.fastIgnorableWhitespace(data); err != nil {
					return err
				}
			} else if s := pctx.sax; s != nil && !pctx.disableSAX {
				if err := pctx.deliverCharacters(ctx, s.IgnorableWhitespace, data); err != nil {
					return err
				}
			}
		} else {
			if pctx.treeBuilder != nil && !pctx.disableSAX {
				if err := pctx.fastCharacters(data); err != nil {
					return err
				}
			} else if s := pctx.sax; s != nil && !pctx.disableSAX {
				if err := pctx.deliverCharacters(ctx, s.Characters, data); err != nil {
					return err
				}
			}
		}
		return nil
	}

	// Fallback: use bytes.Buffer for non-UTF8 cursors.
	buf := bufferPool.Get()
	defer releaseBuffer(buf)

	i := cur.ScanCharDataInto(buf, pctx.nodeContentScanBudget())
	if i <= 0 {
		if cur.Peek() == ']' && cur.PeekAt(1) == ']' && cur.PeekAt(2) == '>' {
			return pctx.error(ctx, ErrMisplacedCDATAEnd)
		}
		return errors.New("invalid char data")
	}
	if pctx.nodeContentTooLong(buf.Len()) {
		return pctx.error(ctx, ErrNodeContentTooLarge)
	}

	if err := cur.AdvanceFast(i); err != nil {
		return err
	}

	data := buf.Bytes()
	if pctx.areBlanksBytes(data, false) {
		if pctx.treeBuilder != nil && !pctx.disableSAX {
			if err := pctx.fastIgnorableWhitespace(data); err != nil {
				return err
			}
		} else if s := pctx.sax; s != nil && !pctx.disableSAX {
			if err := pctx.deliverCharacters(ctx, s.IgnorableWhitespace, data); err != nil {
				return err
			}
		}
	} else {
		if pctx.treeBuilder != nil && !pctx.disableSAX {
			if err := pctx.fastCharacters(data); err != nil {
				return err
			}
		} else if s := pctx.sax; s != nil && !pctx.disableSAX {
			if err := pctx.deliverCharacters(ctx, s.Characters, data); err != nil {
				return err
			}
		}
	}

	return nil
}

// parseCharDataChunkedSAX scans and delivers a character-data run to a streaming
// SAX consumer in chunks of at most pctx.charBufferSize bytes. Unlike the
// single-shot fast path it never materializes the whole delimiter-free run, so a
// huge run delivers with bounded memory. Context cancellation is checked between
// chunks. Used only when charBufferSize > 0 and no DOM is being built (the DOM
// path must classify blank-vs-text over the whole run to drive whitespace
// stripping, so it stays single-shot).
//
// Whitespace classification must match the single-shot path, which classifies
// the WHOLE run as one unit: <root>  text</root> is character data (the leading
// blanks are not ignorable whitespace), while <root>   </root> is ignorable
// whitespace. The chunked path therefore must NOT emit any IgnorableWhitespace
// event until it has proven the whole run is blank — an early per-chunk
// IgnorableWhitespace that a later non-blank byte contradicts cannot be taken
// back. Two cases keep this bounded:
//
//   - When the context makes whitespace non-ignorable here (xml:space="preserve",
//     mixed content, no open element), the run is character data regardless of
//     its bytes, so it is streamed in fixed-size chunks directly. This covers the
//     unbounded-text DoS in those contexts.
//   - Otherwise the leading blank run is accumulated while every byte seen is
//     whitespace. The first non-blank byte proves character data: the accumulated
//     prefix plus the rest of the run is delivered as Characters and the tail is
//     streamed in bounded chunks. A run that stays blank to its end is delivered
//     as IgnorableWhitespace. The realistic huge run (non-blank text) commits to
//     Characters on its first chunk and never accumulates; only a pathological
//     multi-megabyte run of pure whitespace buffers (it cannot be classified
//     without seeing its end), and that is whitespace, not the text DoS vector.
func (pctx *parserCtx) parseCharDataChunkedSAX(ctx context.Context, u8 *strcursor.UTF8Cursor) error {
	s := pctx.sax
	limit := pctx.charBufferSize

	// blankBudget bounds the as-yet-unclassified all-whitespace prefix that may
	// be buffered before it is downgraded to character data. A blank run cannot
	// be proven ignorable whitespace until its end is in view, so without a cap
	// a pathological multi-megabyte run of pure whitespace would accumulate
	// whole in acc before the first callback. The budget is a small multiple of
	// the configured chunk size with a fixed floor, so realistic indentation
	// runs still classify as ignorable whitespace while memory stays bounded.
	blankBudget := max(limit*8, minPendingBlankBytes)

	// blank tracks whether the run could still be ignorable whitespace. When the
	// context makes whitespace non-ignorable, it starts false so the first chunk
	// commits to Characters immediately (no blank-prefix accumulation).
	blank := pctx.whitespaceContextIgnorable()

	acc := pctx.charBuf[:0]
	first := true
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		prev := len(acc)
		var i int
		acc, i = u8.ScanCharDataSlice(acc, limit)
		if i <= 0 {
			if first {
				if u8.Peek() == ']' && u8.PeekAt(1) == ']' && u8.PeekAt(2) == '>' {
					return pctx.error(ctx, ErrMisplacedCDATAEnd)
				}
				return errors.New("invalid char data")
			}
			// The run ended; everything accumulated is blank (a non-blank byte
			// would have returned via the Characters branch below). Deliver it
			// as ignorable whitespace or character data per the final
			// classification below.
			break
		}
		first = false

		if err := u8.AdvanceFast(i); err != nil {
			return err
		}
		pctx.charBuf = acc

		if blank && !allBlankBytes(acc[prev:]) {
			blank = false
		}

		// Bounded-whitespace policy: an unclassified blank prefix that grows
		// past the budget is downgraded to character data so memory stays
		// bounded for a pathological pure-whitespace run.
		//
		// DOCUMENTED POLICY (intentional reclassification, not a silent quirk):
		// an all-whitespace run can only be classified as IgnorableWhitespace
		// once its end-of-run delimiter is in view, so a still-blank prefix must
		// be buffered until then. To bound memory against a pathological multi-MiB
		// run of pure whitespace, once the buffered blank prefix grows past
		// blankBudget we stop treating it as ignorable and deliver it (and the
		// rest of the run) as Characters rather than IgnorableWhitespace. Only
		// abnormally large pure-blank runs are affected — realistic indentation /
		// pretty-printing whitespace is far below blankBudget (see
		// minPendingBlankBytes) and is still delivered as IgnorableWhitespace.
		if blank && len(acc) > blankBudget {
			blank = false
		}

		if !blank {
			// Proven to be (or downgraded to) character data: flush the
			// accumulated prefix (which includes this chunk) as Characters, then
			// stream the rest of the run in bounded chunks.
			if err := pctx.deliverCharacters(ctx, s.Characters, acc); err != nil {
				return err
			}
			return pctx.streamCharDataChunks(ctx, u8, limit, s.Characters)
		}
	}

	pctx.charBuf = acc
	if len(acc) == 0 {
		return nil
	}

	// The run was blank to its end. Match the single-shot areBlanksBytes
	// classification: when no DOM document drives the decision, a blank run is
	// ignorable whitespace only if the delimiter that ended it is '<' or CR. A
	// run ending at '&' (an entity reference) or any other delimiter is
	// character data — the delimiter check whitespaceContextIgnorable omits is
	// re-applied here, now that the end of the run (and thus the delimiter) is
	// in view.
	handler := s.IgnorableWhitespace
	if pctx.doc == nil {
		if c := u8.Peek(); c != '<' && c != 0xD {
			handler = s.Characters
		}
	}
	return pctx.deliverCharacters(ctx, handler, acc)
}

// minPendingBlankBytes is the floor for the blank-prefix budget in
// parseCharDataChunkedSAX: a blank character-data run up to this size is
// buffered and classified as ignorable whitespace even when the configured
// chunk size is tiny, so realistic indentation is never downgraded to
// character data by the bounded-whitespace policy.
const minPendingBlankBytes = 1 << 16 // 64 KiB

// streamCharDataChunks scans the remainder of a character-data run and delivers
// it via handler in chunks of at most limit bytes, with bounded memory. It is
// called once the run's classification is known and at least one chunk has
// already been consumed, so an empty scan means the run ended at a delimiter or
// EOF (handled by the caller) rather than an error.
func (pctx *parserCtx) streamCharDataChunks(ctx context.Context, u8 *strcursor.UTF8Cursor, limit int, handler func(context.Context, []byte) error) error {
	for {
		// A SAX handler may have requested a stop on the previous chunk's
		// callback. Bail before scanning or advancing so no further chunk is
		// emitted after the stop.
		if pctx.stopped {
			return errParserStopped
		}
		if err := ctx.Err(); err != nil {
			return err
		}

		data, i := u8.ScanCharDataSlice(pctx.charBuf[:0], limit)
		if i <= 0 {
			return nil
		}

		if err := u8.AdvanceFast(i); err != nil {
			return err
		}
		pctx.charBuf = data

		if err := pctx.deliverCharacters(ctx, handler, data); err != nil {
			return err
		}
	}
}

func (pctx *parserCtx) parseElement(ctx context.Context) error {
	pctx.elemDepth++
	defer func() { pctx.elemDepth-- }()

	if pctx.maxElemDepth > 0 && pctx.elemDepth > pctx.maxElemDepth {
		return pctx.error(ctx, fmt.Errorf("xml: exceeded max depth"))
	}

	// parseStartTag only parses up to the attributes.
	// For example, given <foo>bar</foo>, the next token would
	// be bar</foo>. Given <foo />, the next token would
	// be />
	if err := pctx.parseStartTag(ctx); err != nil {
		return pctx.error(ctx, err)
	}

	cur := pctx.getCursor()
	if cur == nil {
		return pctx.error(ctx, errNoCursor)
	}
	if cur.Peek() != '/' || cur.PeekAt(1) != '>' {
		if err := pctx.parseContent(ctx); err != nil {
			return pctx.error(ctx, err)
		}
	}

	if err := pctx.parseEndTag(ctx); err != nil {
		return pctx.error(ctx, err)
	}

	return nil
}

// validatePrefixedNamespaceDecl enforces the Namespaces in XML constraints
// that apply to a prefixed namespace declaration (xmlns:prefix="uri"),
// regardless of whether the declaration is literal on a start tag or supplied
// as a DTD attribute default. The reserved xml prefix must map to the XML
// namespace; the xmlns prefix may not be redeclared; the reserved XMLNS
// namespace URI may not be reused; the URI may not be empty; and, in pedantic
// mode, the URI must be absolute. It returns a non-nil namespace error when any
// constraint is violated.
func (pctx *parserCtx) validatePrefixedNamespaceDecl(ctx context.Context, prefix, uri string) error {
	if prefix == lexicon.PrefixXML {
		if uri != lexicon.NamespaceXML {
			return pctx.namespaceError(ctx, errors.New("xml namespace prefix mapped to wrong URI"))
		}
		return nil
	}
	if uri == lexicon.NamespaceXML {
		return pctx.namespaceError(ctx, fmt.Errorf("xmlns:%s: only the xml prefix may be bound to the reserved XML namespace", prefix))
	}
	if prefix == lexicon.PrefixXMLNS {
		return pctx.namespaceError(ctx, errors.New("redefinition of the xmlns prefix forbidden"))
	}
	if uri == lexicon.NamespaceXMLNS {
		return pctx.namespaceError(ctx, errors.New("reuse of the xmlns namespace name if forbidden"))
	}
	if uri == "" {
		// Namespaces in XML 1.1 §5: a prefixed namespace declaration with an
		// empty value undeclares the prefix (removes the in-scope binding).
		// This is well-formed only in an XML 1.1 document; XML 1.0 forbids it.
		// The reserved xml/xmlns prefixes were already handled above, so they
		// cannot be undeclared here.
		if pctx.isXML11() {
			return nil
		}
		return pctx.namespaceError(ctx, fmt.Errorf("xmlns:%s: Empty XML namespace is not allowed", prefix))
	}
	u, err := url.Parse(uri)
	if err != nil {
		return pctx.namespaceError(ctx, fmt.Errorf("xmlns:%s: '%s' is not a validURI", prefix, uri))
	}
	if pctx.pedantic && u.Scheme == "" {
		return pctx.namespaceError(ctx, fmt.Errorf("xmlns:%s: URI %s is not absolute", prefix, uri))
	}
	return nil
}

func (pctx *parserCtx) parseStartTag(ctx context.Context) error {
	cur := pctx.getCursor()
	if cur == nil {
		return pctx.error(ctx, errNoCursor)
	}
	if cur.Peek() != '<' {
		return pctx.error(ctx, ErrStartTagRequired)
	}
	if err := cur.Advance(1); err != nil {
		return err
	}

	local, prefix, err := pctx.parseQName(ctx)
	if local == "" {
		return pctx.error(ctx, fmt.Errorf("local name empty! local = %s, prefix = %s, err = %s", local, prefix, err))
	}
	if err != nil {
		return pctx.error(ctx, err)
	}

	// Push xml:space stack entry for this element (inherit parent's value by default)
	pctx.spaceTab = append(pctx.spaceTab, -1)

	nbNs := 0
	if pctx.attrBuf == nil {
		pctx.attrBuf = make([]attrData, 0, 8)
	}
	attrs := pctx.attrBuf[:0]

	// Prefixes declared by namespace attributes on THIS start tag, used to
	// detect same-element duplicate declarations. The empty string is the
	// default xmlns. This is tracked independently of pushNS/nbNs because
	// parseNsClean may skip pushing a redundant ancestor redeclaration while
	// still having consumed a same-element declaration that a later
	// duplicate must conflict with. Reset per element (nsDeclared[:0]).
	nsDeclared := pctx.nsDeclaredBuf[:0]
	for pctx.instate != psEOF {
		pctx.skipBlanks(ctx)
		if cur.Peek() == '>' {
			if err := cur.Advance(1); err != nil {
				return err
			}
			break
		}

		if cur.Peek() == '/' && cur.PeekAt(1) == '>' {
			break
		}
		attname, aprefix, attvalue, err := pctx.parseAttribute(ctx, local)
		if err != nil {
			return pctx.error(ctx, err)
		}

		if attname == lexicon.PrefixXMLNS && aprefix == "" {
			// <elem xmlns="...">
			// Namespace URI entity/character references are expanded inline
			// during attribute value parsing (replaceEntities forced true in
			// parseAttribute for namespace attrs), so no post-processing needed.

			// A start tag may not carry two default namespace declarations.
			// This is a well-formedness violation regardless of whether the
			// URIs match and is fatal even with parseNsClean (which only
			// suppresses redundant ancestor redeclarations, never
			// same-element duplicates). The check uses nsDeclared, which
			// records every same-element declaration even one the parseNsClean
			// skip path would not push onto nsTab.
			if slices.Contains(nsDeclared, "") {
				return pctx.error(ctx, errors.New("duplicate attribute is not allowed"))
			}
			nsDeclared = append(nsDeclared, "")

			// parseNsClean: skip redundant ancestor redeclarations.
			if pctx.options.IsSet(parseNsClean) && pctx.nsTab.Lookup("") == attvalue {
				goto SkipDefaultNS
			}
			pctx.pushNS("", attvalue)
			nbNs++
		SkipDefaultNS:
			if cur.Peek() == '>' || (cur.Peek() == '/' && cur.PeekAt(1) == '>') {
				continue
			}

			if !isBlankByte(cur.Peek()) {
				return pctx.error(ctx, ErrSpaceRequired)
			}
			pctx.skipBlanks(ctx)
			continue
		} else if aprefix == lexicon.PrefixXMLNS {
			// <elem xmlns:foo="...">
			// Namespace URI entity/character references are expanded inline
			// during attribute value parsing (replaceEntities forced true in
			// parseAttribute for namespace attrs), so no post-processing needed.
			// The same validity checks are applied to DTD-defaulted namespace
			// declarations during attribute defaulting below.
			if err := pctx.validatePrefixedNamespaceDecl(ctx, attname, attvalue); err != nil {
				return err
			}
			if attname == lexicon.PrefixXML {
				// Record the explicitly-declared reserved prefix before the
				// SkipNS shortcut so a conflicting DTD-supplied default for the
				// same prefix (e.g. <!ATTLIST r xmlns:xml CDATA "urn:dtd">) is
				// suppressed by the nsDeclared check during attribute
				// defaulting. Without this, the explicit binding takes the early
				// goto and is never recorded, letting the DTD default override
				// the reserved xml namespace.
				nsDeclared = append(nsDeclared, attname)
				goto SkipNS
			}

			// A same-element duplicate namespace declaration is a
			// well-formedness violation and is fatal even when the URIs
			// match and parseNsClean is set: parseNsClean only suppresses
			// redundant ancestor redeclarations, never same-element dupes.
			// nsDeclared records every same-element declaration, including
			// one the parseNsClean skip path would not push onto nsTab, so a
			// later duplicate is still caught. A prefix bound only in an
			// ancestor is valid shadowing and is not in nsDeclared.
			if slices.Contains(nsDeclared, attname) {
				return pctx.error(ctx, errors.New("duplicate attribute is not allowed"))
			}
			nsDeclared = append(nsDeclared, attname)
			// parseNsClean: skip if an ancestor already binds this prefix
			// to the same URI (redundant redeclaration).
			if pctx.options.IsSet(parseNsClean) && pctx.nsTab.Lookup(attname) == attvalue {
				goto SkipNS
			}
			pctx.pushNS(attname, attvalue)
			nbNs++

		SkipNS:
			if cur.Peek() == '>' || (cur.Peek() == '/' && cur.PeekAt(1) == '>') {
				continue
			}

			if !isBlankByte(cur.Peek()) {
				return pctx.error(ctx, ErrSpaceRequired)
			}
			pctx.skipBlanks(ctx)
			continue
		}

		// XML 1.0 §3.1: a start tag may not carry two attributes with the
		// same qualified name. Reject before appending or invoking any
		// SAX/DOM callback. (Namespace declarations are duplicate-checked
		// in their own branches above and never reach here.)
		for i := range attrs {
			if attrs[i].localname == attname && attrs[i].prefix == aprefix {
				return pctx.error(ctx, errors.New("duplicate attribute is not allowed"))
			}
		}

		attr := attrData{
			localname: attname,
			prefix:    aprefix,
			value:     attvalue,
		}

		attrs = append(attrs, attr)
	}

	// Attributes defaulting: apply DTD-declared default attribute values.
	// NOTE: #FIXED/#REQUIRED validation and element content model checking
	// are done post-parse via validateDocument() when parseDTDValid is set.
	// ID/IDREF uniqueness checks are done post-parse via validateDocument().
	if len(pctx.attsDefault) > 0 {
		var elemName string
		if prefix != "" {
			elemName = prefix + ":" + local
		} else {
			elemName = local
		}

		defaults, ok := pctx.lookupAttributeDefault(elemName)
		if ok {
			// First pass: apply default xmlns="..." (must come before prefixed).
			// Skip a DTD default whose prefix (the empty string for the default
			// namespace) was already explicitly declared on this start tag: an
			// explicit binding must win over a DTD-supplied default. Because
			// nsStack.Lookup is LIFO, pushing the DTD default afterwards would
			// otherwise shadow the explicit one.
			for _, attr := range defaults {
				if attr.LocalName() == lexicon.PrefixXMLNS && attr.Prefix() == "" {
					if slices.Contains(nsDeclared, "") {
						continue
					}
					pctx.pushNS("", attr.Value())
					nbNs++
				}
			}
			// Second pass: apply xmlns:prefix="..." and regular attributes.
			// Likewise skip a prefixed DTD default already declared explicitly.
			for _, attr := range defaults {
				attname := attr.LocalName()
				aprefix := attr.Prefix()
				if attname == lexicon.PrefixXMLNS && aprefix == "" {
					continue
				} else if aprefix == lexicon.PrefixXMLNS {
					if slices.Contains(nsDeclared, attname) {
						continue
					}
					// DTD-defaulted namespace declarations are subject to the
					// same namespace-validity checks as literal ones: a
					// wrong-URI xmlns:xml, an xmlns:xmlns redefinition, reuse of
					// the reserved XMLNS namespace, an empty/invalid URI, etc.
					// are all rejected before the binding is pushed.
					if err := pctx.validatePrefixedNamespaceDecl(ctx, attname, attr.Value()); err != nil {
						return err
					}
					// The reserved xml prefix is implicitly bound to the XML
					// namespace; never let a DTD default push (and thus shadow)
					// it, even a well-formed one. This mirrors the literal path,
					// where xmlns:xml takes the SkipNS shortcut without pushing.
					if attname == lexicon.PrefixXML {
						continue
					}
					pctx.pushNS(attname, attr.Value())
					nbNs++
				} else {
					dup := false
					for _, ea := range attrs {
						if ea.localname == attname && ea.prefix == aprefix {
							dup = true
							break
						}
					}
					if !dup {
						attrs = append(attrs, attrData{
							localname: attname,
							prefix:    aprefix,
							value:     attr.Value(),
							isDefault: attr.IsDefault(),
						})
					}
				}
			}
		}
	}

	for _, a := range attrs {
		if a.prefix == lexicon.PrefixXML && a.localname == "space" {
			switch a.value {
			case "preserve":
				pctx.spaceTab[len(pctx.spaceTab)-1] = 1
			case "default":
				pctx.spaceTab[len(pctx.spaceTab)-1] = 0
			}
			break
		}
	}

	// Namespaces in XML §6.3: no element may have two attributes with the
	// same expanded name (namespace URI + local name). Literal duplicate
	// qualified names were already rejected during parsing; here we catch
	// the case of distinct prefixes bound to the same namespace URI
	// (e.g. p:a and q:a where xmlns:p and xmlns:q both map to urn:x).
	// This is done after all namespace declarations on this start tag have
	// been pushed, so prefixes declared after the attributes still resolve.
	// Unprefixed attributes are in no namespace (a default xmlns does not
	// apply to attributes) and are excluded from this check.
	for i := range attrs {
		if attrs[i].prefix == "" || attrs[i].prefix == lexicon.PrefixXML {
			continue
		}
		iuri := pctx.lookupNamespace(attrs[i].prefix)
		for j := i + 1; j < len(attrs); j++ {
			if attrs[j].prefix == "" || attrs[j].prefix == lexicon.PrefixXML {
				continue
			}
			if attrs[i].localname != attrs[j].localname {
				continue
			}
			if iuri != "" && iuri == pctx.lookupNamespace(attrs[j].prefix) {
				return pctx.error(ctx, errors.New("duplicate attribute is not allowed"))
			}
		}
	}

	nsuri := pctx.lookupNamespace(prefix)
	if prefix != "" && nsuri == "" {
		return pctx.namespaceError(ctx, errors.New("namespace '"+prefix+"' not found"))
	}

	if pctx.treeBuilder != nil && !pctx.disableSAX {
		if err := pctx.fastStartElement(local, prefix, nsuri, attrs, nbNs); err != nil {
			return pctx.error(ctx, err)
		}
	} else if s := pctx.sax; s != nil && !pctx.disableSAX {
		var nslist []sax.Namespace
		if nbNs > 0 {
			nslist = make([]sax.Namespace, nbNs)
			for i, ns := range pctx.nsTab.Peek(nbNs) {
				nslist[i] = ns
			}
		}
		var saxAttrs []sax.Attribute
		if len(attrs) > 0 {
			saxAttrs = make([]sax.Attribute, len(attrs))
			for i := range attrs {
				saxAttrs[i] = &attrs[i]
			}
		}
		switch err := s.StartElementNS(ctx, local, prefix, nsuri, nslist, saxAttrs); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return pctx.error(ctx, err)
		}
	}
	qname := local
	if prefix != "" {
		qname = prefix + ":" + local
	}
	pctx.pushNodeEntry(nodeEntry{local: local, prefix: prefix, uri: nsuri, qname: qname})
	pctx.nsNrTab = append(pctx.nsNrTab, nbNs)
	pctx.attrBuf = attrs[:0]
	pctx.nsDeclaredBuf = nsDeclared[:0]

	return nil
}

/**
 * parse an end of tag
 *
 * [42] ETag ::= '</' Name S? '>'
 *
 * With namespace
 *
 * [NS 9] ETag ::= '</' QName S? '>'
 */
func (pctx *parserCtx) parseEndTag(ctx context.Context) error {
	cur := pctx.getCursor()
	if cur == nil {
		return pctx.error(ctx, errNoCursor)
	}
	if cur.Peek() == '/' && cur.PeekAt(1) == '>' {
		if err := cur.Advance(2); err != nil {
			return err
		}
	} else {
		if !cur.ConsumeString("</") {
			return pctx.error(ctx, ErrLtSlashRequired)
		}

		e := pctx.peekNode()
		if !cur.ConsumeString(e.Name()) {
			return pctx.error(ctx, errors.New("expected end tag '"+e.Name()+"'"))
		}

		pctx.skipBlanks(ctx)

		if cur.Peek() != '>' {
			return pctx.error(ctx, ErrGtRequired)
		}
		if err := cur.Advance(1); err != nil {
			return err
		}
	}

	e := pctx.peekNode()
	if pctx.treeBuilder != nil && !pctx.disableSAX {
		if err := pctx.fastEndElement(); err != nil {
			return pctx.error(ctx, err)
		}
	} else if s := pctx.sax; s != nil && !pctx.disableSAX {
		switch err := s.EndElementNS(ctx, e.LocalName(), e.Prefix(), e.URI()); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return pctx.error(ctx, err)
		}
	}
	pctx.popNode()

	if len(pctx.spaceTab) > 1 {
		pctx.spaceTab = pctx.spaceTab[:len(pctx.spaceTab)-1]
	}

	if n := len(pctx.nsNrTab); n > 0 {
		nbNs := pctx.nsNrTab[n-1]
		pctx.nsNrTab = pctx.nsNrTab[:n-1]
		if nbNs > 0 {
			pctx.nsTab.Pop(nbNs)
		}
	}

	return nil
}

func (pctx *parserCtx) parseAttributeValue(ctx context.Context, normalize bool) (value string, entities int, err error) {
	cur := pctx.getCursor()
	if cur == nil {
		err = pctx.error(ctx, errNoCursor)
		return
	}
	qch := cur.Peek()
	switch qch {
	case '"', '\'':
		if err = cur.Advance(1); err != nil {
			return
		}
	default:
		err = errors.New("string not started (got '" + string([]byte{qch}) + "')")
		return
	}

	value, entities, err = pctx.parseAttributeValueInternal(ctx, qch, normalize)
	if err != nil {
		return
	}

	if cur.Peek() != qch {
		err = errors.New("string not closed")
		return
	}
	err = cur.Advance(1)
	return
}

// This is based on xmlParseAttValueComplex
func (pctx *parserCtx) parseAttributeValueInternal(ctx context.Context, qch byte, normalize bool) (value string, entities int, err error) {
	prevState := pctx.instate
	pctx.instate = psAttributeValue
	defer func() { pctx.instate = prevState }()

	cur := pctx.getCursor()
	if cur == nil {
		err = pctx.error(ctx, errNoCursor)
		return
	}

	if !normalize {
		if u8, ok := cur.(*strcursor.UTF8Cursor); ok {
			if v, nBytes := u8.ScanSimpleAttrValue(qch, pctx.nodeContentScanBudget()); nBytes > 0 {
				// The scan budget is cap+utf8.UTFMax, so a successful scan can
				// run slightly over the cap; re-check the exact byte count here
				// (before advancing) so a value of cap+1..cap+UTFMax bytes is
				// rejected, matching the slow path's per-iteration check.
				if pctx.nodeContentTooLong(nBytes) {
					err = pctx.error(ctx, ErrNodeContentTooLarge)
					return
				}
				if err = u8.AdvanceFast(nBytes); err != nil {
					return
				}
				value = v
				return
			}
		}
	}

	inSpace := false
	b := bufferPool.Get()
	defer releaseBuffer(b)

	// Every write into b goes through writeAttr{String,Byte,Rune}, which
	// enforce the node-content cap BEFORE the copy. This bounds the value
	// during accumulation (so a giant attribute fails before its closing quote
	// is reached, matching CDATA/PI/comment) AND closes the whole class of
	// single-iteration over-cap writes — most importantly the non-substituted
	// entity-reference branch, which copies "&"+ent.name+";" in one step and
	// would otherwise be unbounded for a long entity name under MaxNameLength(-1).
	for {
		c := cur.PeekRune()
		if (qch != 0x0 && c == rune(qch)) || c == '<' {
			break
		}
		// Width-aware char validation: a real U+FFFD is encoded as valid
		// 3-byte UTF-8 and is a legal XML Char, whereas invalid/incomplete
		// UTF-8 decodes to RuneError with width 1. isChar rejects every
		// RuneError, so decode with width here to tell the two apart and
		// keep the slow path consistent with the fast path.
		dr, dw, ok := decodeRuneAt(cur, 0)
		if !ok {
			break
		}
		if dr == utf8.RuneError && dw == 1 {
			break
		}
		if !xmlchar.IsChar(dr) {
			break
		}
		switch c {
		case '&':
			entities++
			inSpace = false
			if cur.PeekAt(1) == '#' {
				var r rune
				r, err = pctx.parseCharRef()
				if err != nil {
					err = pctx.error(ctx, err)
					return
				}

				if r == '&' && !pctx.replaceEntities {
					if err = pctx.writeAttrString(ctx, b, "&#38;"); err != nil {
						return
					}
				} else {
					if err = pctx.writeAttrRune(ctx, b, r); err != nil {
						return
					}
				}
			} else {
				var ent *Entity
				ent, err = pctx.parseEntityRef(ctx)
				if err != nil {
					err = pctx.error(ctx, err)
					return
				}

				if ent == nil {
					continue
				}

				if ent.entityType == enum.InternalPredefinedEntity {
					if ent.content == "&" && !pctx.replaceEntities {
						if err = pctx.writeAttrString(ctx, b, "&#38;"); err != nil {
							return
						}
					} else {
						if err = pctx.writeAttrString(ctx, b, ent.content); err != nil {
							return
						}
					}
				} else if pctx.replaceEntities {
					// Decode the entity replacement DIRECTLY into the attribute
					// buffer through a cap-enforcing sink instead of first
					// materializing the full expansion via decodeEntities and
					// then copying it in. The sink normalizes attribute-value
					// whitespace (TAB/CR/LF -> space) and checks the node-content
					// cap before every byte, so an over-cap expansion (e.g.
					// <r a="&big;"/> with SubstituteEntities, or a
					// forced-replacement namespace attr xmlns:x="&big;") fails
					// with ErrNodeContentTooLarge as soon as the running total
					// would exceed the remaining budget — the cap is enforced
					// incrementally during decode, never after a fully-built rep.
					sink := &attrEntitySink{pctx: pctx, b: b}
					if err = pctx.decodeEntitiesToSink(ctx, ent.Content(), SubstituteRef, 0, sink); err != nil {
						err = pctx.error(ctx, err)
						return
					}
				} else {
					// Attribute-value WFCs on the entity's TRANSITIVE replacement
					// text ("No External Entity References", "No < in Attribute
					// Values"). This is consulted on EVERY occurrence, independent
					// of ent.checked: the checked bit records only the weaker
					// general-content (element) check, so an entity first expanded
					// in element content must still be re-validated under the
					// stricter attribute-value rules here. The predicate fires no
					// SAX callbacks, so it does not duplicate ResolveEntity events.
					switch pctx.attributeEntityWFC(ent) {
					case attrWFCExternal:
						err = pctx.error(ctx, errors.New("attribute references external entity"))
						return
					case attrWFCUnparsed:
						err = pctx.error(ctx, errors.New("entity reference to unparsed entity"))
						return
					case attrWFCLessThan:
						err = pctx.error(ctx, errors.New("'<' in entity is not allowed in attribute values"))
						return
					}
					if ent.checked == 0 && strings.ContainsRune(ent.content, '&') {
						// Validate the entity's replacement text: a general entity
						// referenced from an attribute value must expand to
						// well-formed content, so an undefined entity nested in it
						// (WFC: Entity Declared) is a fatal error, not something to
						// swallow. (W3C not-wf-sa-077.)
						if _, derr := pctx.decodeEntities(ctx, ent.Content(), SubstituteRef); derr != nil {
							err = pctx.error(ctx, derr)
							return
						}
						ent.checked = 2
					}
					// Route the unresolved reference through the bounded helper:
					// a declared entity with a very long name under
					// MaxNameLength(-1) would otherwise copy "&"+ent.name+";"
					// unbounded in this single iteration before the next cap
					// check.
					if err = pctx.writeAttrString(ctx, b, "&"); err != nil {
						return
					}
					if err = pctx.writeAttrString(ctx, b, ent.name); err != nil {
						return
					}
					if err = pctx.writeAttrString(ctx, b, ";"); err != nil {
						return
					}
				}
			}
		case 0x20, 0xD, 0xA, 0x9:
			if b.Len() > 0 || !normalize {
				if !normalize || !inSpace {
					if err = pctx.writeAttrByte(ctx, b, 0x20); err != nil {
						return
					}
				}
				inSpace = true
			}
			if err := cur.Advance(1); err != nil {
				return "", 0, err
			}
		default:
			inSpace = false
			// Write the raw decoded bytes (dw wide) so a real U+FFFD round-trips
			// intact; WriteRune(c) would re-encode RuneError and utf8.RuneLen(c)
			// would be -1, advancing too few bytes.
			if err = pctx.writeAttrString(ctx, b, cur.PeekString(dw)); err != nil {
				return
			}
			if err := cur.Advance(dw); err != nil {
				return "", 0, err
			}
		}
	}

	value = b.String()
	if inSpace && normalize {
		if value[len(value)-1] == 0x20 {
			for len(value) > 0 {
				if value[len(value)-1] != 0x20 {
					break
				}
				value = value[:len(value)-1]
			}
		}
	}

	return
}

// attributeEntityWFC reports whether the general entity ent, referenced from an
// attribute value under SubstituteEntities(false), violates one of the XML 1.0
// attribute-value well-formedness constraints via its TRANSITIVE replacement
// text: "No External Entity References" (a nested external or unparsed general
// entity) or "No < in Attribute Values" (a nested literal '<'). The DIRECT case
// (ent itself external/unparsed, or its own content directly containing '<') is
// caught earlier by parseEntityRef; this covers an entity that only reaches such
// content through nested &name; references and whose weaker general-content check
// (ent.checked) must not suppress the stricter attribute-value WFCs.
//
// The result is memoized on ent: it is a pure function of the entity tables,
// which do not change mid-parse. Crucially the walk resolves nested references
// with pctx.getEntity (a plain table lookup that fires NO SAX callbacks), unlike
// decodeEntities, so consulting it on every occurrence never duplicates
// ResolveEntity events.
func (pctx *parserCtx) attributeEntityWFC(ent *Entity) attrEntityWFC {
	if ent.attrWFCSet {
		return ent.attrWFC
	}
	visited := map[*Entity]struct{}{ent: {}}
	res := pctx.scanAttrEntityWFC(ent.content, visited)
	ent.attrWFC = res
	ent.attrWFCSet = true
	return res
}

// scanAttrEntityWFC walks replacement text for a literal '<' or a nested general
// reference to an external/unparsed entity, recursing through internal general
// entities. visited guards against entity reference cycles. Predefined entities
// (&lt; &gt; &amp; &apos; &quot;) are the sanctioned escapes and are never a
// violation, matching parseEntityRef's InternalPredefinedEntity exclusion.
func (pctx *parserCtx) scanAttrEntityWFC(content string, visited map[*Entity]struct{}) attrEntityWFC {
	for i := 0; i < len(content); i++ {
		c := content[i]
		if c == '<' {
			return attrWFCLessThan
		}
		if c != '&' {
			continue
		}
		semi := strings.IndexByte(content[i+1:], ';')
		if semi < 0 {
			break
		}
		ref := content[i+1 : i+1+semi]
		i += 1 + semi // loop's i++ then moves past ';'
		if len(ref) == 0 || ref[0] == '#' {
			// Char reference: character data. Any '<' it resolves to is an
			// allowed escape (&#60;), so it is intentionally not flagged.
			continue
		}
		nested, err := pctx.getEntity(ref)
		if err != nil || nested == nil {
			// Undefined nested entity: the "Entity Declared" WFC is enforced by
			// the decodeEntities validation, not here — do not double-report.
			continue
		}
		switch nested.entityType {
		case enum.ExternalGeneralUnparsedEntity:
			return attrWFCUnparsed
		case enum.ExternalGeneralParsedEntity:
			return attrWFCExternal
		case enum.InternalGeneralEntity:
			if _, seen := visited[nested]; seen {
				continue
			}
			visited[nested] = struct{}{}
			if v := pctx.scanAttrEntityWFC(nested.content, visited); v != attrWFCNone {
				return v
			}
		}
	}
	return attrWFCNone
}

func (pctx *parserCtx) parseAttribute(ctx context.Context, elemName string) (local string, prefix string, value string, err error) {
	l, p, err := pctx.parseQName(ctx)
	if err != nil {
		err = pctx.error(ctx, err)
		return
	}

	normalize := false
	attType, ok := pctx.lookupSpecialAttribute(elemName, l)
	if ok && attType != enum.AttrInvalid {
		normalize = true
	}
	// xml:id normalization (xml:id Recommendation §4 + XML §3.3.3 tokenized-type
	// normalization): an xml:id attribute is implicitly xs:ID, so its value is
	// trimmed and internal space runs are collapsed even with NO DTD declaration.
	// This is a DELIBERATE XPath-3.1 / xml:id-§4 conformance choice, NOT libxml2
	// parity: libxml2 normalizes xml:id ONLY when it is DTD-declared ID and leaves
	// undeclared-xml:id normalization as a documented open issue (it does not do
	// it). Verified to leave every libxml2-compat / c14n / serialization golden
	// byte-identical (no parity fixture carries a normalizable-whitespace xml:id).
	// The normalized value is what GetElementByID / fn:id / the XPath string-value
	// of the attribute observe.
	if p == lexicon.PrefixXML && l == "id" {
		normalize = true
	}
	pctx.skipBlanks(ctx)

	cur := pctx.getCursor()
	if cur == nil {
		err = pctx.error(ctx, errNoCursor)
		return
	}
	if cur.Peek() != '=' {
		err = pctx.error(ctx, ErrEqualSignRequired)
		return
	}
	if err := cur.Advance(1); err != nil {
		return "", "", "", err
	}
	pctx.skipBlanks(ctx)

	isNamespace := (l == lexicon.PrefixXMLNS && p == "") || p == lexicon.PrefixXMLNS
	savedReplaceEntities := pctx.replaceEntities
	if isNamespace {
		pctx.replaceEntities = true
	}

	v, entities, err := pctx.parseAttributeValue(ctx, normalize)

	pctx.replaceEntities = savedReplaceEntities

	if err != nil {
		err = pctx.error(ctx, err)
		return
	}

	if normalize {
		if entities > 0 {
			v = pctx.attrNormalizeSpace(v)
		}
	}

	local = l
	prefix = p
	value = v
	err = nil
	return
}

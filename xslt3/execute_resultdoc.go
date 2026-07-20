package xslt3

import (
	"context"
	"errors"
	"maps"
	"net/url"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/sequence"
	"github.com/lestrrat-go/helium/xpath3"
)

// canonicalResultURIKey produces a canonical key for XTDE1490 duplicate
// detection. Unlike helium.BuildURI (which strips the file: scheme for file:
// bases, turning "file:///d/out.xml" into "/d/out.xml"), this resolves the
// href as a URI and PRESERVES the scheme/host for both absolute and relative
// hrefs. That ensures a relative href ("out.xml") and the equivalent absolute
// href ("file:///d/out.xml") under the same base collapse to the same key, so
// two result documents denoting the same file are detected as duplicates.
func canonicalResultURIKey(href, base string) string {
	ref, err := url.Parse(href)
	if err != nil {
		return href
	}
	if ref.IsAbs() {
		// Collapse "." / ".." dot-segments so that
		// "file:///base/dir/a/../out.xml" and "file:///base/dir/out.xml" produce
		// the SAME key (XTDE1490 duplicate detection). Resolving an absolute
		// hierarchical reference against itself runs RFC 3986 remove_dot_segments
		// while preserving the scheme/authority.
		return ref.ResolveReference(ref).String()
	}
	if base == "" {
		return ref.ResolveReference(ref).String()
	}
	baseURL, err := url.Parse(base)
	if err != nil || !baseURL.IsAbs() {
		// Without a usable absolute base we cannot canonicalize URI-wise;
		// fall back to helium.BuildURI's filesystem-style resolution so that
		// distinct relative hrefs denoting the same path still collide.
		if resolved := helium.BuildURI(href, base); resolved != "" {
			return resolved
		}
		return href
	}
	return baseURL.ResolveReference(ref).String()
}

// paramDocPresence records which plain-boolean serialization parameters a
// parameter-document explicitly supplied. These flags are deliberately kept off
// the public OutputDef: a plain bool cannot distinguish "omitted" from "explicit
// false", and foldParamDocOverrides consults them so an omitted parameter-document
// value leaves an inherited xsl:output default intact instead of clobbering it
// with the Go zero value. They travel alongside the parameter-document delta
// OutputDef (on the compiled instruction or the per-invocation cache).
type paramDocPresence struct {
	indent              bool
	byteOrderMark       bool
	allowDuplicateNames bool
	undeclarePrefixes   bool
}

// getParamDocOutputDef returns the effective parameter-document OutputDef for
// a result-document instruction, checking the per-invocation cache on
// execContext first, then falling back to the compiled instruction's field.
func (ec *execContext) getParamDocOutputDef(inst *resultDocumentInst) *OutputDef {
	if od, ok := ec.paramDocOutputDefs[inst]; ok {
		return od
	}
	return inst.ParameterDocOutputDef
}

// getParamDocPresence returns the plain-boolean presence flags that accompany
// the effective parameter-document delta, mirroring getParamDocOutputDef's
// cache-then-instruction lookup so the runtime AVT and compile-time static
// parameter-document paths stay in sync.
func (ec *execContext) getParamDocPresence(inst *resultDocumentInst) paramDocPresence {
	if p, ok := ec.paramDocPresences[inst]; ok {
		return p
	}
	return inst.ParameterDocPresence
}

// validateDocumentStructure checks that a document node has exactly one element
// child, no text nodes (non-whitespace), and only comments/PIs otherwise.
// Returns XTTE1550 on violation.
func validateDocumentStructure(doc *helium.Document) error {
	elemCount := 0
	for child := range helium.Children(doc) {
		switch child.Type() {
		case helium.ElementNode:
			elemCount++
		case helium.TextNode, helium.CDATASectionNode:
			// Any text content at the document level (including whitespace
			// that is not whitespace-only) fails validation.
			text := strings.TrimSpace(string(child.Content()))
			if text != "" {
				return dynamicError(errCodeXTTE1550,
					"validated document has text nodes at the top level")
			}
		case helium.CommentNode, helium.ProcessingInstructionNode, helium.DTDNode:
			// Allowed at document level.
		}
	}
	if elemCount != 1 {
		return dynamicError(errCodeXTTE1550,
			"validated document must have exactly one root element, found %d", elemCount)
	}
	return nil
}

// moveChildren transfers every child of src into dst, preserving order. Each
// child is unlinked from src before being added to dst so that the source
// document's sibling/parent links do not remain attached to the moved nodes
// (which would alias nodes across two trees and can corrupt sibling traversal).
func moveChildren(src *helium.Document, dst *helium.Document) error {
	var children []helium.Node
	for child := range helium.Children(src) {
		children = append(children, child)
	}
	for _, child := range children {
		helium.UnlinkNode(child.(helium.MutableNode)) //nolint:forcetypeassert
		if err := dst.AddChild(child); err != nil {
			return err
		}
	}
	return nil
}

// applyValidationResult records the schema type annotations and nilled
// elements produced by a ValidateDoc call onto the execution context.
func (ec *execContext) applyValidationResult(vr validateDocResult) {
	for node, typeName := range vr.Annotations {
		ec.annotateNode(node, typeName)
	}
	for elem := range vr.NilledElements {
		ec.markNilled(elem)
	}
}

// execDocument implements xsl:document: creates a document node wrapping
// the result of executing the body.
func (ec *execContext) execDocument(ctx context.Context, inst *documentInst) error {
	tmpDoc := helium.NewDefaultDocument()
	frame := &outputFrame{doc: tmpDoc, current: tmpDoc}
	ec.outputStack = append(ec.outputStack, frame)
	if err := ec.executeSequenceConstructor(ctx, inst.Body); err != nil {
		ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
		return err
	}
	ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]

	// Apply type validation (xsl:document type="...").
	if inst.TypeName != "" {
		// XTTE1550: document must have exactly one element child.
		if err := validateDocumentStructure(tmpDoc); err != nil {
			return err
		}
		if ec.schemaRegistry != nil {
			root := findDocumentElement(tmpDoc)
			if root != nil {
				if err := ec.validateAndNormalizeElementContent(ctx, root, inst.TypeName); err != nil {
					if xsltErr, ok := errors.AsType[*XSLTError](err); ok && xsltErr.Code == errCodeXTTE1510 {
						return dynamicError(errCodeXTTE1540,
							"document content does not match declared type %s: %v", inst.TypeName, xsltErr.Message)
					}
					return err
				}
				// Annotate the root element with the validated type so that
				// subsequent instance-of checks can see it.
				ec.annotateNode(root, inst.TypeName)
			}
		}
	}

	// Apply validation if requested (xsl:document validation="strict"|"lax").
	if v := inst.Validation; v == validationStrict || v == validationLax {
		if v == validationStrict {
			if err := validateDocumentStructure(tmpDoc); err != nil {
				return err
			}
		}
		if ec.schemaRegistry != nil {
			vr, valErr := ec.schemaRegistry.ValidateDoc(ctx, tmpDoc)
			if valErr != nil && v == validationStrict {
				return dynamicError(errCodeXTTE1510, "validation of document node failed: %v", valErr)
			}
			if valErr == nil && v == validationStrict {
				// XTTE1510: strict validation requires a matching schema.
				// If no annotations were produced, no schema matched.
				if len(vr.Annotations) == 0 {
					root := findDocumentElement(tmpDoc)
					rootNS := ""
					if root != nil {
						rootNS = root.URI()
					}
					return dynamicError(errCodeXTTE1510,
						"no matching schema declaration for document element in namespace %q", rootNS)
				}
				// XTTE1555: check xs:ID uniqueness and xs:IDREF resolution.
				if err := validateDocIDConstraints(tmpDoc, vr.Annotations); err != nil {
					return err
				}
			}
			ec.applyValidationResult(vr)
		}
	}

	// When the temporary document has no children (e.g. empty TVT body),
	// the xsl:document node still acts as a text-node boundary that breaks
	// the atomic adjacency chain (XSLT 3.0 §5.7.2).
	// In sequence/capture mode the empty document node must still be emitted
	// as an item so it is visible to the caller.
	out := ec.currentOutput()
	if tmpDoc.FirstChild() == nil {
		if out.sequenceMode || (out.captureItems && out.current != nil && out.current.Type() != helium.DocumentNode) {
			out.pendingItems = append(out.pendingItems, xpath3.NodeItem{Node: tmpDoc})
			out.noteOutput()
			return nil
		}
		if !out.wherePopulated {
			out.prevWasAtomic = false
		}
		return nil
	}

	// Emit the document node as an item in the parent output frame.
	// sequenceMode means we are in evaluateBodyAsSequence — emit as item.
	// captureItems with a non-document insertion point means simple content
	// construction (e.g., inside xsl:comment) — emit the document as a
	// single item so that atomization yields the correct string value
	// (excluding comment nodes). wherePopulated means we are inside
	// xsl:where-populated — emit the document node so emptiness can be
	// checked. Otherwise copy children directly.
	if out.sequenceMode || (out.captureItems && out.current != nil && out.current.Type() != helium.DocumentNode) {
		out.pendingItems = append(out.pendingItems, xpath3.NodeItem{Node: tmpDoc})
		out.noteOutput()
	} else if out.wherePopulated {
		if err := ec.addNode(tmpDoc); err != nil {
			return err
		}
	} else {
		// Move children from tmpDoc to the parent output. When validation
		// produced type annotations (strict, lax, preserve, or type="..."),
		// move nodes directly (unlink + addNode) so that annotations keyed
		// by node pointer are preserved.
		preserveAnnotations := inst.Validation == validationPreserve || inst.Validation == validationStrict || inst.Validation == validationLax || inst.TypeName != ""
		if preserveAnnotations {
			var children []helium.Node
			for child := range helium.Children(tmpDoc) {
				children = append(children, child)
			}
			for _, child := range children {
				helium.UnlinkNode(child.(helium.MutableNode)) //nolint:forcetypeassert
				if err := ec.addNode(child); err != nil {
					return err
				}
			}
		} else {
			for child := range helium.Children(tmpDoc) {
				if err := ec.copyNodeToOutput(child); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// validateJSONItems checks for SERE0022 (duplicate keys) in JSON-serializable items.
func validateJSONItems(items xpath3.Sequence) error {
	for item := range sequence.Items(items) {
		if m, ok := item.(xpath3.MapItem); ok {
			if err := validateMapDuplicateKeys(m); err != nil {
				return err
			}
		}
		if a, ok := item.(xpath3.ArrayItem); ok {
			for _, member := range a.Members() {
				if err := validateJSONItems(member); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// validateMapDuplicateKeys checks a map for keys that produce the same
// string representation, which is a SERE0022 error in JSON serialization.
func validateMapDuplicateKeys(m xpath3.MapItem) error {
	seenKeys := make(map[string]struct{})
	var dupErr error
	_ = m.ForEach(func(k xpath3.AtomicValue, v xpath3.Sequence) error {
		if dupErr != nil {
			return dupErr
		}
		ks, _ := xpath3.AtomicToString(k)
		if _, dup := seenKeys[ks]; dup {
			dupErr = dynamicError(errCodeSERE0022, "duplicate key %q in JSON output", ks)
			return dupErr
		}
		seenKeys[ks] = struct{}{}
		// Recursively check nested maps/arrays.
		if err := validateJSONItems(v); err != nil {
			dupErr = err
			return err
		}
		return nil
	})
	return dupErr
}

// isItemOutputMethod returns true when the current effective output method
// supports non-node items (maps, arrays, function items).
func (ec *execContext) isItemOutputMethod() bool {
	return isItemSerializationMethod(ec.currentResultDocMethod)
}

// resolveResultDocFormat returns the effective format name for a result-document
// instruction, evaluating the format avt if present.
// Returns an error for XTDE0290 (prefix not bound) or XTDE1460 (invalid QName).
func (ec *execContext) resolveResultDocFormat(ctx context.Context, inst *resultDocumentInst) (string, error) {
	if inst.FormatAVT != nil {
		v, err := inst.FormatAVT.evaluate(ctx, ec.contextNode)
		if err != nil {
			// A dynamic error raised while evaluating the format AVT must be
			// surfaced, not swallowed: silently falling back to the static
			// format would hide a transformation failure from the caller.
			return "", err
		}
		v = strings.TrimSpace(v)
		if v != "" && !strings.HasPrefix(v, "Q{") {
			// XTDE0290: prefix must have a namespace binding
			if prefix, _, ok := strings.Cut(v, ":"); ok {
				if _, ok := inst.NSBindings[prefix]; !ok {
					return "", dynamicError(errCodeXTDE0290,
						"prefix %q in result-document format has no namespace binding", prefix)
				}
			}
		}
		return resolveQName(v, inst.NSBindings), nil
	}
	return inst.Format, nil
}

// resolveResultDocMethod returns the effective output method for a result-document
// instruction, considering the method avt, compile-time method, named format, and
// default output definition.
func (ec *execContext) resolveResultDocMethod(ctx context.Context, inst *resultDocumentInst) string {
	// Runtime avt takes priority.
	if inst.MethodAVT != nil {
		v, err := inst.MethodAVT.evaluate(ctx, ec.contextNode)
		if err == nil {
			return strings.TrimSpace(v)
		}
	}
	// Compile-time method attribute (may have been set from parameter-document).
	if inst.Method != "" {
		return inst.Method
	}
	// Parameter-document output definition.
	if pd := ec.getParamDocOutputDef(inst); pd != nil && pd.Method != "" {
		return pd.Method
	}
	// Named format.
	format, _ := ec.resolveResultDocFormat(ctx, inst)
	if format != "" {
		if outDef, ok := ec.effectiveOutputs()[format]; ok {
			return outDef.Method
		}
	}
	// Default output definition.
	if outDef, ok := ec.effectiveOutputs()[""]; ok {
		return outDef.Method
	}
	return methodXML
}

// isItemSerializationMethod returns true when the output method supports
// non-node items (maps, arrays, function items) without XTDE0450.
func isItemSerializationMethod(method string) bool {
	return method == methodJSON || method == methodAdaptive
}

// commitPrimaryOutputState publishes the serialization overrides and
// character-map state that a primary xsl:result-document contributes to the
// final primary output. It is the single commit point shared by EVERY primary
// sub-branch (default direct-write, build-tree="no", type="...",
// validation="strict|lax"): each branch evaluates the preflight overrides up
// front (before touching the primary tree) and calls this only after its body
// and post-body checks succeed. Calling it uniformly guarantees that the
// serialization AVTs and character maps declared on a primary result-document
// take effect regardless of which validation/type/build-tree branch handled it.
func (ec *execContext) commitPrimaryOutputState(inst *resultDocumentInst, effectiveFormat string, primaryOverrides *OutputDef) {
	// Propagate character map names to the primary output frame.
	// Include maps from the named format (xsl:output) first, then
	// maps from xsl:result-document itself (higher priority).
	var allMaps []string
	if effectiveFormat != "" {
		if fmtDef, ok := ec.effectiveOutputs()[effectiveFormat]; ok {
			allMaps = append(allMaps, fmtDef.UseCharacterMaps...)
			// Also propagate resolved character maps from parameter-document.
			// Clone the compiled map so the later maps.Copy merge below never
			// mutates the compiled format's ResolvedCharMap.
			if len(fmtDef.ResolvedCharMap) > 0 {
				ec.primaryResolvedCharMap = maps.Clone(fmtDef.ResolvedCharMap)
			}
		}
	}
	allMaps = append(allMaps, inst.UseCharacterMaps...)
	if len(allMaps) > 0 {
		ec.primaryCharacterMaps = allMaps
		// Resolve character maps now while currentPackage is correct
		// (package-scoped isolation). Merge into primaryResolvedCharMap.
		resolved := resolveCharacterMaps(ec.effectiveStylesheet(), allMaps)
		if len(resolved) > 0 {
			if ec.primaryResolvedCharMap == nil {
				ec.primaryResolvedCharMap = resolved
			} else {
				maps.Copy(ec.primaryResolvedCharMap, resolved)
			}
		}
	}
	// Capture serialization parameter overrides from xsl:result-document.
	// These were already evaluated up front (before any primary output was
	// emitted) so that an AVT error releases the URI reservation without
	// leaving partial primary output behind.
	if primaryOverrides != nil {
		ec.primaryOutputOverrides = primaryOverrides
	}
}

func (ec *execContext) execResultDocument(ctx context.Context, inst *resultDocumentInst) error {
	// XTDE1480: xsl:result-document is not allowed in a temporary output state.
	if ec.temporaryOutputDepth > 0 {
		return dynamicError(errCodeXTDE1480, "xsl:result-document is not allowed while in temporary output state")
	}

	// Evaluate the href avt to determine the output URI.
	href := ""
	if inst.Href != nil {
		var err error
		href, err = inst.Href.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
	}

	isPrimary := href == ""

	// XTDE1490 duplicate detection keys on the canonical (resolved) output URI,
	// not the raw href: distinct hrefs that resolve to the same absolute URI
	// (e.g. "a/../out.xml" and "out.xml" under the same base) target the same
	// document and must collide. The primary output (empty href) always keys on
	// "" so that the implicit-primary claim tracked elsewhere stays consistent.
	dupKey := href
	if !isPrimary {
		dupKey = canonicalResultURIKey(href, ec.currentOutputURI)
	}
	if _, used := ec.usedResultURIs[dupKey]; used {
		return dynamicError(errCodeXTDE1490, "two result documents written to same URI: %q", dupKey)
	}

	if isPrimary && ec.primaryClaimedImplicitly {
		return dynamicError(errCodeXTRE1495, "primary output URI already has implicit content")
	}

	// Reserve the canonical URI so a concurrent/nested result-document targeting
	// the same URI collides, but treat the reservation as provisional until the
	// result document is actually committed. Any error before commit (e.g. a
	// format AVT that raises a dynamic error, a failed parameter-document load,
	// or a body that throws) must release the reservation. Otherwise an
	// xsl:result-document caught inside xsl:try would leave its URI permanently
	// claimed, making an xsl:catch that writes the same href fail with a
	// spurious XTDE1490 even though no result document was ever written there.
	ec.usedResultURIs[dupKey] = struct{}{}
	committed := false
	defer func() {
		if !committed {
			delete(ec.usedResultURIs, dupKey)
		}
	}()

	// Resolve the effective format name (static or avt).
	effectiveFormat, fmtErr := ec.resolveResultDocFormat(ctx, inst)
	if fmtErr != nil {
		return fmtErr
	}

	// XTDE1460: the format attribute must reference a declared xsl:output.
	if effectiveFormat != "" {
		if _, ok := ec.effectiveOutputs()[effectiveFormat]; !ok {
			return dynamicError(errCodeXTDE1460,
				"xsl:result-document format %q does not match any declared xsl:output", effectiveFormat)
		}
	}

	// Resolve parameter-document if specified as avt. Store the resolved
	// output def on execContext (not on the compiled instruction) to avoid
	// mutating the stylesheet's instruction tree.
	if inst.ParameterDocAVT != nil && inst.ParameterDocOutputDef == nil {
		if _, cached := ec.paramDocOutputDefs[inst]; !cached {
			pdHref, pdErr := inst.ParameterDocAVT.evaluate(ctx, ec.contextNode)
			if pdErr != nil {
				return pdErr
			}
			if pdHref != "" {
				outDef := &OutputDef{}
				baseURI := ec.effectiveStaticBaseURI()
				// runtime=true so the helper classifies a load/parse failure as
				// a dynamic error (FODC0002), never a static one. A failure must
				// not be silently dropped: the transformation has to fail so
				// callers can observe it, and an over-cap read keeps
				// [ErrResourceTooLarge] in the chain (matched via errors.Is)
				// while errors.Is(err, ErrStaticError) stays false.
				_, presence, loadErr := loadParameterDocumentFromFile(ctx, ec.injectedParser(), outDef, baseURI, pdHref, ec.retrieveDocumentBytes, true, false, ec.resourceLimit())
				if loadErr != nil {
					return loadErr
				}
				if ec.paramDocOutputDefs == nil {
					ec.paramDocOutputDefs = make(map[*resultDocumentInst]*OutputDef)
				}
				ec.paramDocOutputDefs[inst] = outDef
				if ec.paramDocPresences == nil {
					ec.paramDocPresences = make(map[*resultDocumentInst]paramDocPresence)
				}
				ec.paramDocPresences[inst] = presence
			}
		}
	}

	// Resolve effective item-separator: xsl:result-document attribute takes
	// priority (including #absent which blocks format inheritance),
	// then the named xsl:output (format), then nil (default).
	var itemSep *string
	if inst.ItemSeparatorSet {
		// Attribute was present on xsl:result-document; evaluate avt value
		// (nil for #absent, or the evaluated string).
		if inst.ItemSeparator != nil {
			sepVal, err := inst.ItemSeparator.evaluate(ctx, ec.contextNode)
			if err != nil {
				return err
			}
			itemSep = &sepVal
		}
	} else if effectiveFormat != "" {
		if outDef, ok := ec.effectiveOutputs()[effectiveFormat]; ok && outDef.ItemSeparator != nil {
			itemSep = outDef.ItemSeparator
		}
	} else if outDef, ok := ec.effectiveOutputs()[""]; ok && outDef.ItemSeparator != nil {
		itemSep = outDef.ItemSeparator
	}

	// Track the current output URI for current-output-uri().
	// For secondary outputs, resolve relative href against the current output URI
	// using the SAME URI-preserving resolver as the XTDE1490 duplicate key
	// (canonicalResultURIKey). helium.BuildURI strips the file: scheme for file:
	// bases, so a nested result-document inside a secondary output would compute a
	// scheme-stripped base ("/base/dir/inner.xml") that no longer matches the
	// duplicate key's scheme-preserving form ("file:///base/dir/inner.xml"),
	// causing a relative href and its absolute file: equivalent to miss inside the
	// secondary output. Canonicalizing here keeps both forms consistent.
	savedOutputURI := ec.currentOutputURI
	if href != "" {
		ec.currentOutputURI = canonicalResultURIKey(href, savedOutputURI)
	}
	// For primary output (href==""), currentOutputURI stays unchanged.
	defer func() { ec.currentOutputURI = savedOutputURI }()

	if isPrimary {
		// PREFLIGHT (uniform across EVERY primary sub-branch): evaluate the
		// error-prone serialization parameter AVTs (method, standalone, indent,
		// doctype, omit-xml-declaration, etc.) BEFORE any branch executes its body
		// or mutates/emits primary output. This MUST run above the early type=/
		// validation= returns: those branches previously returned before the
		// preflight, so a serialization AVT that raises a dynamic error (e.g.
		// standalone="{1 idiv 0}") was silently swallowed and primaryOutputOverrides/
		// character-map state was never applied for them. Computing the overrides up
		// front guarantees that any such error happens before the primary output is
		// touched: the deferred cleanup releases the URI reservation so an xsl:catch
		// may write the primary result document, and no partial primary output is
		// left behind. The staged overrides are committed (via commitPrimaryOutputState)
		// only after each branch's body and post-body checks succeed.
		primaryOverrides, err := ec.evalResultDocOutputDef(ctx, inst)
		if err != nil {
			return err
		}
		effectiveMethod := ec.resolveResultDocMethod(ctx, inst)

		v := inst.Validation
		if inst.TypeName != "" && v == "" {
			// type attribute without explicit validation: build into temp doc, validate type, copy.
			tmpDoc := helium.NewDefaultDocument()
			savedMethod := ec.currentResultDocMethod
			ec.currentResultDocMethod = effectiveMethod
			ec.outputStack = append(ec.outputStack, &outputFrame{doc: tmpDoc, current: tmpDoc, itemSeparator: itemSep})
			if err := ec.executeSequenceConstructor(ctx, inst.Body); err != nil {
				ec.currentResultDocMethod = savedMethod
				ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
				return err
			}
			ec.currentResultDocMethod = savedMethod
			ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
			root := findDocumentElement(tmpDoc)
			if root != nil && ec.schemaRegistry != nil {
				if err := ec.validateAndNormalizeElementContent(ctx, root, inst.TypeName); err != nil {
					if xsltErr, ok := errors.AsType[*XSLTError](err); ok && xsltErr.Code == errCodeXTTE1510 {
						return dynamicError(errCodeXTTE1540,
							"result document content does not match declared type %s: %v", inst.TypeName, xsltErr.Message)
					}
					return err
				}
			}
			primaryFrame := ec.outputStack[0]
			if err := moveChildren(tmpDoc, primaryFrame.doc); err != nil {
				return err
			}
			ec.commitPrimaryOutputState(inst, effectiveFormat, primaryOverrides)
			committed = true
			return nil
		}
		if v == validationStrict || v == validationLax {
			// When validation is requested for the primary output, build into a
			// temporary document, validate it, then copy children to the primary
			// output. This is the only way we can inspect the complete document
			// structure before emitting it.
			tmpDoc := helium.NewDefaultDocument()
			savedMethod := ec.currentResultDocMethod
			ec.currentResultDocMethod = effectiveMethod
			ec.outputStack = append(ec.outputStack, &outputFrame{doc: tmpDoc, current: tmpDoc, itemSeparator: itemSep})
			if err := ec.executeSequenceConstructor(ctx, inst.Body); err != nil {
				ec.currentResultDocMethod = savedMethod
				ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
				return err
			}
			ec.currentResultDocMethod = savedMethod
			ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
			// XTTE1550: validate document structure.
			if v == validationStrict {
				if err := validateDocumentStructure(tmpDoc); err != nil {
					return err
				}
			}
			if ec.schemaRegistry != nil {
				vr, valErr := ec.schemaRegistry.ValidateDoc(ctx, tmpDoc)
				if valErr != nil && v == validationStrict {
					return dynamicError(errCodeXTTE1540, "validation of primary result document failed: %v", valErr)
				}
				if valErr == nil && v == validationStrict {
					// XTTE1555: check xs:ID uniqueness and xs:IDREF resolution.
					if err := validateDocIDConstraints(tmpDoc, vr.Annotations); err != nil {
						return err
					}
				}
				ec.applyValidationResult(vr)
			}
			// Move validated children into the primary output.
			primaryFrame := ec.outputStack[0]
			if err := moveChildren(tmpDoc, primaryFrame.doc); err != nil {
				return err
			}
			ec.commitPrimaryOutputState(inst, effectiveFormat, primaryOverrides)
			committed = true
			return nil
		}
		// Drive build-tree from the EVALUATED effective output def (primaryOverrides),
		// not a static compile-time bool: build-tree may be an AVT (e.g.
		// build-tree="{false()}") and may also be inherited from the named format or
		// parameter-document folded into the overrides. primaryOverrides is nil only
		// when no serialization params (build-tree included) were set, in which case
		// build-tree defaults to true.
		buildTreeNo := primaryOverrides != nil && primaryOverrides.BuildTree != nil && !*primaryOverrides.BuildTree

		// When build-tree="no", execute into a temporary document,
		// then extract children and pending items as a raw sequence
		// for serialization with item-separator.
		if buildTreeNo && isItemSerializationMethod(effectiveMethod) {
			tmpDoc := helium.NewDefaultDocument()
			tmpRoot, err := tmpDoc.CreateElement("_tmp")
			if err != nil {
				return err
			}
			if err := tmpDoc.AddChild(tmpRoot); err != nil {
				return err
			}
			savedMethod := ec.currentResultDocMethod
			ec.currentResultDocMethod = effectiveMethod
			// Use sequenceMode + captureItems to capture ALL items
			// (elements, comments, maps, attributes) in order.
			ec.outputStack = append(ec.outputStack, &outputFrame{
				doc: tmpDoc, current: tmpRoot,
				itemSeparator: itemSep,
				captureItems:  true,
				sequenceMode:  true,
			})
			ec.insideResultDocPrimary = true
			if err := ec.executeSequenceConstructor(ctx, inst.Body); err != nil {
				ec.insideResultDocPrimary = false
				ec.currentResultDocMethod = savedMethod
				ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
				return err
			}
			frame := ec.outputStack[len(ec.outputStack)-1]
			ec.insideResultDocPrimary = false
			ec.currentResultDocMethod = savedMethod
			ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]

			out := ec.outputStack[0]
			// SERE0022: validate JSON duplicate keys when allow-duplicate-names
			// is not "yes". This branch selects JSON purely from the
			// result-document's effective method, so currentResultDocMethod has
			// already been restored by the time the final primary validation runs
			// in execute_transform.go; validate here against the preflighted
			// overrides instead. Mirrors the buffered direct-write path.
			if effectiveMethod == methodJSON {
				allowDupes := false
				if primaryOverrides != nil {
					allowDupes = primaryOverrides.AllowDuplicateNames
				} else if defDef, ok := ec.effectiveOutputs()[""]; ok {
					allowDupes = defDef.AllowDuplicateNames
				}
				if !allowDupes {
					if err := validateJSONItems(frame.pendingItems); err != nil {
						return err
					}
				}
			}
			out.pendingItems = append(out.pendingItems, frame.pendingItems...)

			ec.commitPrimaryOutputState(inst, effectiveFormat, primaryOverrides)
			committed = true
			return nil
		}

		// Buffer the primary direct-write path: execute the body into a
		// temporary frame and splice it into the real primary output ONLY after
		// the body and all post-body checks succeed. A body that throws (e.g.
		// inside xsl:try) must leave NO partial primary output behind, so that
		// the deferred release of the "" reservation is sound and an xsl:catch
		// that writes another primary result document does not produce a
		// double-primary result.
		//
		// The buffer frame stands IN for the base frame: it temporarily replaces
		// ec.outputStack[0] (rather than being pushed on top), so that the body
		// observes the same outputStack depth (1) and the same insideResultDocPrimary
		// state as a true direct write. This keeps every depth-sensitive branch
		// (XTRE1495 suppression, rawResultSequence capture, etc.) behaving exactly
		// as before; only the destination tree changes, and only until commit.
		realBase := ec.outputStack[0]
		bufDoc := helium.NewDefaultDocument()
		// Clone the REAL base frame so every output flag (captureItems,
		// sequenceMode, documentConstructor, mapConstructor, wherePopulated,
		// separateTextNodes, …) and accumulator (prevWasAtomic, prevHadOutput,
		// outputSerial, …) is preserved. If the real primary frame captures items
		// (raw delivery via cfg.rawCapture, or a json/adaptive default output
		// method), atomics from xsl:sequence MUST be preserved as XDM items, not
		// stringified into the DOM. Override ONLY the per-buffer destination
		// (doc/current), the per-buffer item-separator, and the per-buffer
		// accumulators that must start fresh for this buffer (pendingItems,
		// seqPlaceholders, conditionalScopes). Everything else carries over.
		bufFrameVal := *realBase
		bufFrame := &bufFrameVal
		bufFrame.doc = bufDoc
		bufFrame.current = bufDoc
		bufFrame.itemSeparator = itemSep
		bufFrame.pendingItems = nil
		bufFrame.seqPlaceholders = nil
		bufFrame.conditionalScopes = nil
		savedStack := ec.outputStack
		bufferedStack := make([]*outputFrame, len(ec.outputStack))
		copy(bufferedStack, ec.outputStack)
		bufferedStack[0] = bufFrame
		ec.outputStack = bufferedStack[:1] // keep only the (buffered) base frame
		ec.insideResultDocPrimary = true
		savedMethod := ec.currentResultDocMethod
		ec.currentResultDocMethod = effectiveMethod
		savedRawResult := ec.rawResultSequence
		if err := ec.executeSequenceConstructor(ctx, inst.Body); err != nil {
			ec.insideResultDocPrimary = false
			ec.currentResultDocMethod = savedMethod
			ec.rawResultSequence = savedRawResult
			ec.outputStack = savedStack
			return err
		}
		// Validate JSON duplicate keys (SERE0022) when allow-duplicate-names is not "yes".
		if effectiveMethod == methodJSON {
			// allow-duplicate-names defaults to "no" per XSLT 3.0 §20. The value
			// (including the result-document AVT and any named-format/default
			// xsl:output base) was already evaluated up front in the preflight via
			// evalResultDocOutputDef; reuse it so a failing AVT was surfaced there
			// rather than silently swallowed here.
			// When the primary result-document declares no serialization
			// attributes of its own, primaryOverrides is nil. In that case the
			// effective allow-duplicate-names comes from the resolved default
			// output definition (e.g. a stylesheet-level
			// <xsl:output method="json" allow-duplicate-names="yes"/>), not a
			// hard-coded "no".
			allowDupes := false
			if primaryOverrides != nil {
				allowDupes = primaryOverrides.AllowDuplicateNames
			} else if defDef, ok := ec.effectiveOutputs()[""]; ok {
				allowDupes = defDef.AllowDuplicateNames
			}
			if !allowDupes {
				if err := validateJSONItems(bufFrame.pendingItems); err != nil {
					ec.insideResultDocPrimary = false
					ec.currentResultDocMethod = savedMethod
					ec.rawResultSequence = savedRawResult
					ec.outputStack = savedStack
					return err
				}
			}
		}
		ec.insideResultDocPrimary = false
		ec.currentResultDocMethod = savedMethod
		ec.outputStack = savedStack
		// Body and all post-body checks succeeded: splice the buffered content
		// into the real primary output and propagate the accumulator state.
		if err := moveChildren(bufDoc, realBase.doc); err != nil {
			return err
		}
		realBase.pendingItems = append(realBase.pendingItems, bufFrame.pendingItems...)
		realBase.prevWasAtomic = bufFrame.prevWasAtomic
		realBase.prevHadOutput = bufFrame.prevHadOutput
		realBase.outputSerial = bufFrame.outputSerial
		realBase.emptyAtomicGen = bufFrame.emptyAtomicGen
		realBase.seqConstructorGen = bufFrame.seqConstructorGen
		// Commit the serialization overrides + character-map state. The overrides
		// were evaluated up front (before any primary output was emitted) so that
		// an AVT error releases the URI reservation without leaving partial primary
		// output behind.
		ec.commitPrimaryOutputState(inst, effectiveFormat, primaryOverrides)
		committed = true
		return nil
	}

	// Secondary output: execute body into a temporary document.
	tmpDoc := helium.NewDefaultDocument()

	// Set the document URL so that base-uri() returns the correct value. Per XSLT
	// 3.0 §26.2 a secondary result document's base URI is its href resolved
	// against the BASE OUTPUT URI, NOT the stylesheet base URI. ec.currentOutputURI
	// already holds exactly that value: it was set above (canonicalResultURIKey of
	// href against the saved base output URI) and is the same scheme-preserving
	// canonical URI used as the XTDE1490 duplicate-detection key.
	tmpDoc.SetURL(ec.currentOutputURI)

	// PREFLIGHT (symmetric with the primary path): evaluate the secondary
	// effective output definition — which evaluates EVERY error-prone
	// serialization parameter AVT (method, standalone, indent, doctype, etc.) via
	// evalResultDocOutputDef — BEFORE running the body. A serialization AVT that
	// raises a dynamic error (e.g. method="{1 idiv 0}") inside xsl:try must roll
	// the transaction back with NO body executed and NO per-href side effect: were
	// the AVTs evaluated AFTER the body (the prior order), a NESTED secondary
	// result-document inside the body could commit before the outer instruction's
	// AVT error surfaced, leaving the enclosing xsl:catch to observe a stale nested
	// result document. Building the output def up front guarantees any such error
	// happens before inst.Body executes.
	stagedOutDef, err := ec.buildEffectiveOutputDef(ctx, inst, effectiveFormat, "")
	if err != nil {
		return err
	}

	// Derive the effective output method from the preflighted output definition so
	// a MethodAVT error is NOT swallowed (resolveResultDocMethod silently ignores
	// MethodAVT failures): buildEffectiveOutputDef already surfaced any AVT error
	// above. Fall back to resolveResultDocMethod only for the non-AVT resolution
	// chain (static method, parameter-document, named format, default).
	effectiveMethod := stagedOutDef.Method
	if effectiveMethod == "" {
		effectiveMethod = ec.resolveResultDocMethod(ctx, inst)
		stagedOutDef.Method = effectiveMethod
		stagedOutDef.MethodExplicit = true
	}

	savedMethod := ec.currentResultDocMethod
	ec.currentResultDocMethod = effectiveMethod
	captureSecondary := isItemSerializationMethod(effectiveMethod)
	ec.outputStack = append(ec.outputStack, &outputFrame{doc: tmpDoc, current: tmpDoc, itemSeparator: itemSep, captureItems: captureSecondary})
	if err := ec.executeSequenceConstructor(ctx, inst.Body); err != nil {
		ec.currentResultDocMethod = savedMethod
		ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
		return err
	}
	frame := ec.outputStack[len(ec.outputStack)-1]
	ec.currentResultDocMethod = savedMethod
	ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]

	// Stage ALL per-href state locally; publish into the shared evaluator maps
	// only at the single commit point below, after the body AND every post-body
	// validation step has succeeded. If a validation error is thrown here (and
	// possibly caught by an enclosing xsl:try), NO per-href side effect may
	// persist: a stale resultDocItems/resultDocOutputDefs entry would otherwise
	// be materialized at end-of-transform or contaminate an xsl:catch that writes
	// the same href (spurious XTDE1490 / wrong output). The three staged pieces
	// are: the captured json/adaptive items, the effective output definition (built
	// up front in the preflight above), and the result-document tree itself.
	var stagedItems xpath3.Sequence
	if isItemSerializationMethod(effectiveMethod) && len(frame.pendingItems) > 0 {
		stagedItems = frame.pendingItems
	}

	// Apply type validation for secondary result documents (type="...").
	if inst.TypeName != "" && inst.Validation == "" {
		root := findDocumentElement(tmpDoc)
		if root != nil && ec.schemaRegistry != nil {
			if err := ec.validateAndNormalizeElementContent(ctx, root, inst.TypeName); err != nil {
				if xsltErr, ok := errors.AsType[*XSLTError](err); ok && xsltErr.Code == errCodeXTTE1510 {
					return dynamicError(errCodeXTTE1540,
						"result document content does not match declared type %s: %v", inst.TypeName, xsltErr.Message)
				}
				return err
			}
			ec.annotateNode(root, inst.TypeName)
		}
	}

	// Validate the result document if requested.
	if v := inst.Validation; v == validationStrict || v == validationLax {
		// XTTE1550: when validating a document node, the children must comprise
		// exactly one element node, no text nodes, and zero or more comment and
		// processing instruction nodes, in any order.
		if v == validationStrict {
			if err := validateDocumentStructure(tmpDoc); err != nil {
				return err
			}
		}
		if ec.schemaRegistry != nil {
			vr, valErr := ec.schemaRegistry.ValidateDoc(ctx, tmpDoc)
			if valErr != nil && v == validationStrict {
				return dynamicError(errCodeXTTE1510, "validation of result document failed: %v", valErr)
			}
			if valErr == nil && v == validationStrict {
				// Check that the root element has a matching schema declaration.
				if len(vr.Annotations) == 0 {
					root := findDocumentElement(tmpDoc)
					if root != nil {
						rootLocal := root.LocalName()
						rootNS := root.URI()
						if _, found := ec.schemaRegistry.LookupElement(rootLocal, rootNS); !found {
							return dynamicError(errCodeXTTE1510,
								"no matching schema declaration for element {%s}%s in result document (validation=strict)", rootNS, rootLocal)
						}
					}
				}
				// XTTE1555: check xs:ID uniqueness and xs:IDREF resolution.
				if err := validateDocIDConstraints(tmpDoc, vr.Annotations); err != nil {
					return err
				}
			}
			ec.applyValidationResult(vr)
		}
	} else if inst.Validation == validationStrip {
		root := findDocumentElement(tmpDoc)
		if root != nil {
			ec.stripAnnotations(root)
		}
	}

	// SERE0022: a SECONDARY (href) JSON result document must reject duplicate
	// keys unless allow-duplicate-names="yes", exactly like the primary path. The
	// final SerializeItems pass in execute_transform.go does NOT re-validate
	// staged secondary items, so without this check a secondary
	// method="json" build-tree="no" result document would silently accept
	// map{1:'a','1':'b'}. Validate against the preflighted output definition
	// before committing so a thrown SERE0022 leaves no per-href side effect.
	if effectiveMethod == methodJSON && stagedItems != nil && !stagedOutDef.AllowDuplicateNames {
		if err := validateJSONItems(stagedItems); err != nil {
			return err
		}
	}

	// COMMIT POINT: body and every post-body validation step succeeded. Publish
	// all staged per-URI state atomically, then mark the transaction committed so
	// the deferred rollback leaves the usedResultURIs reservation in place. Storage
	// keys on the RESOLVED canonical absolute output URI (dupKey == tmpDoc.URL() ==
	// ec.currentOutputURI), NOT the raw href: two NESTED xsl:result-documents with
	// the same relative href under different enclosing output URIs resolve to
	// distinct absolute URIs and must not overwrite each other. The raw href is
	// preserved separately for the public ResultDocumentHandler.
	if stagedItems != nil {
		ec.resultDocItems[dupKey] = stagedItems
	}
	if stagedOutDef != nil {
		ec.resultDocOutputDefs[dupKey] = stagedOutDef
	}
	ec.resultDocuments[dupKey] = tmpDoc
	ec.resultDocHrefs[dupKey] = href
	committed = true
	return nil
}

// evalBoolSerializationAVT evaluates a boolean serialization-parameter AVT on
// xsl:result-document and returns its xs:boolean value. A non-empty value that
// is not a valid xs:boolean lexical form raises SEPM0016 instead of being
// silently coerced to false. This is the SINGLE chokepoint that EVERY boolean
// serialization parameter (indent, omit-xml-declaration, byte-order-mark,
// escape-uri-attributes, include-content-type, allow-duplicate-names) must route
// through, so the invalid-value validation can never drift parameter-by-parameter
// again. (standalone is a tri-state yes/no/omit value and is handled separately.)
func (ec *execContext) evalBoolSerializationAVT(ctx context.Context, a *avt, paramName string) (bool, error) {
	v, err := a.evaluate(ctx, ec.contextNode)
	if err != nil {
		return false, err
	}
	b, ok := parseXSDBool(strings.TrimSpace(v))
	if !ok {
		return false, dynamicError(errCodeSEPM0016,
			"%q is not a valid value for xsl:result-document/@%s", v, paramName)
	}
	return b, nil
}

// foldParamDocOverrides overlays the serialization parameters supplied by a
// result-document's parameter-document (pd, loaded as a delta from an empty
// OutputDef) onto base, which already holds the inherited named-format or
// unnamed-default xsl:output values. A parameter-document outranks the format it
// layers on, so any value it specifies wins; values it OMITS leave base intact.
// String/pointer/slice fields are absent when zero/nil/empty; the plain-boolean
// parameters rely on the companion paramDocPresence flags because a false bool
// cannot otherwise be distinguished from an omitted one.
func foldParamDocOverrides(base, pd *OutputDef, pres paramDocPresence) {
	if pd.Method != "" {
		base.Method = pd.Method
		base.MethodExplicit = pd.MethodExplicit
	}
	if pd.Encoding != "" {
		base.Encoding = pd.Encoding
	}
	if pd.Standalone != "" {
		base.Standalone = pd.Standalone
	}
	if len(pd.CDATASections) > 0 {
		base.CDATASections = append([]string(nil), pd.CDATASections...)
	}
	if pd.DoctypePublic != "" {
		base.DoctypePublic = pd.DoctypePublic
	}
	if pd.DoctypeSystem != "" {
		base.DoctypeSystem = pd.DoctypeSystem
	}
	if pd.MediaType != "" {
		base.MediaType = pd.MediaType
	}
	if pd.Version != "" {
		base.Version = pd.Version
	}
	if pd.NormalizationForm != "" {
		base.NormalizationForm = pd.NormalizationForm
	}
	if pd.HTMLVersion != "" {
		base.HTMLVersion = pd.HTMLVersion
	}
	if pd.JSONNodeOutputMethod != "" {
		base.JSONNodeOutputMethod = pd.JSONNodeOutputMethod
	}
	if len(pd.SuppressIndentation) > 0 {
		base.SuppressIndentation = append([]string(nil), pd.SuppressIndentation...)
	}
	if pd.IncludeContentType != nil {
		v := *pd.IncludeContentType
		base.IncludeContentType = &v
	}
	if pd.EscapeURIAttributes != nil {
		v := *pd.EscapeURIAttributes
		base.EscapeURIAttributes = &v
	}
	if pd.BuildTree != nil {
		v := *pd.BuildTree
		base.BuildTree = &v
	}
	if pd.ItemSeparator != nil {
		v := *pd.ItemSeparator
		base.ItemSeparator = &v
	}
	if len(pd.ResolvedCharMap) > 0 {
		base.ResolvedCharMap = make(map[rune]string, len(pd.ResolvedCharMap))
		maps.Copy(base.ResolvedCharMap, pd.ResolvedCharMap)
	}
	if pd.OmitDeclarationExplicit {
		base.OmitDeclaration = pd.OmitDeclaration
		base.OmitDeclarationExplicit = true
	}
	if pres.indent {
		base.Indent = pd.Indent
	}
	if pres.byteOrderMark {
		base.ByteOrderMark = pd.ByteOrderMark
	}
	if pres.allowDuplicateNames {
		base.AllowDuplicateNames = pd.AllowDuplicateNames
	}
	if pres.undeclarePrefixes {
		base.UndeclarePrefixes = pd.UndeclarePrefixes
	}
}

// evalResultDocOutputDef evaluates serialization parameter AVTs on
// xsl:result-document and returns an OutputDef with the overrides.
// Returns nil if no serialization parameters are specified.
func (ec *execContext) evalResultDocOutputDef(ctx context.Context, inst *resultDocumentInst) (*OutputDef, error) {
	hasAny := inst.MethodAVT != nil || inst.Standalone != nil || inst.Indent != nil ||
		inst.OmitXMLDeclaration != nil || inst.DoctypeSystem != nil || inst.DoctypePublic != nil ||
		inst.CDATASectionElements != nil || inst.Encoding != nil || inst.OutputVersion != nil ||
		inst.ByteOrderMark != nil || inst.EscapeURIAttributes != nil ||
		inst.MediaType != nil || inst.HTMLVersion != nil || inst.IncludeContentType != nil ||
		inst.AllowDuplicateNames != nil || inst.UndeclarePrefixes != nil ||
		inst.JSONNodeOutputMethodAVT != nil || inst.NormalizationForm != nil ||
		ec.getParamDocOutputDef(inst) != nil ||
		inst.ItemSeparatorSet || inst.BuildTree != nil ||
		len(inst.SuppressIndentation) > 0
	effectiveFormat, fmtErr := ec.resolveResultDocFormat(ctx, inst)
	if fmtErr != nil {
		return nil, fmtErr
	}
	if !hasAny && effectiveFormat == "" {
		return nil, nil //nolint:nilnil
	}

	// Build the effective base in serialization-parameter priority order
	// (low -> high): the named format (or the unnamed default xsl:output when no
	// format), then the parameter-document. A parameter-document has HIGHER
	// priority than the format it layers on, so its values win; values it omits
	// must leave the inherited format/default intact. The earlier implementation
	// took the parameter-document OutputDef as the whole base, which dropped the
	// unnamed-default xsl:output entirely: a parameter-document that omitted a
	// plain boolean (indent, byte-order-mark, allow-duplicate-names,
	// undeclare-prefixes) then overwrote an inherited true with the Go zero value
	// (wrongly disabling inherited indent/BOM, or rejecting duplicate JSON keys a
	// default allow-duplicate-names="yes" permits). Folding the default base first
	// and overlaying only the parameter-document's set values fixes that and keeps
	// the primary-output direct-assign in execute_transform.go correct.
	var base OutputDef
	paramDocOD := ec.getParamDocOutputDef(inst)
	if effectiveFormat != "" {
		if fmtDef, ok := ec.effectiveOutputs()[effectiveFormat]; ok {
			base = *cloneOutputDef(fmtDef)
		}
	} else if defDef, ok := ec.effectiveOutputs()[""]; ok {
		base = *cloneOutputDef(defDef)
	}
	if paramDocOD != nil {
		foldParamDocOverrides(&base, paramDocOD, ec.getParamDocPresence(inst))
	}

	evalAVT := func(avt *avt) (string, error) {
		if avt == nil {
			return "", nil
		}
		return avt.evaluate(ctx, ec.contextNode)
	}

	if inst.MethodAVT != nil {
		v, err := evalAVT(inst.MethodAVT)
		if err != nil {
			return nil, err
		}
		base.Method = strings.TrimSpace(v)
		base.MethodExplicit = true
	}
	if inst.Standalone != nil {
		v, err := evalAVT(inst.Standalone)
		if err != nil {
			return nil, err
		}
		switch strings.TrimSpace(v) {
		case lexicon.ValueTrue, "1":
			v = lexicon.ValueYes
		case "false", "0":
			v = lexicon.ValueNo
		default:
			v = strings.TrimSpace(v)
		}
		base.Standalone = v
	}
	if inst.Indent != nil {
		b, err := ec.evalBoolSerializationAVT(ctx, inst.Indent, paramIndent)
		if err != nil {
			return nil, err
		}
		base.Indent = b
	}
	if inst.OmitXMLDeclaration != nil {
		b, err := ec.evalBoolSerializationAVT(ctx, inst.OmitXMLDeclaration, paramOmitXMLDeclaration)
		if err != nil {
			return nil, err
		}
		base.OmitDeclaration = b
		// The result-document explicitly specified omit-xml-declaration, so the
		// xhtml/html5 serializer must not flip it back to "yes" by default.
		base.OmitDeclarationExplicit = true
	}
	if inst.DoctypeSystem != nil {
		v, err := evalAVT(inst.DoctypeSystem)
		if err != nil {
			return nil, err
		}
		base.DoctypeSystem = v
	}
	if inst.DoctypePublic != nil {
		v, err := evalAVT(inst.DoctypePublic)
		if err != nil {
			return nil, err
		}
		base.DoctypePublic = v
	}
	if inst.Encoding != nil {
		v, err := evalAVT(inst.Encoding)
		if err != nil {
			return nil, err
		}
		base.Encoding = strings.TrimSpace(v)
	}
	if inst.OutputVersion != nil {
		v, err := evalAVT(inst.OutputVersion)
		if err != nil {
			return nil, err
		}
		base.Version = strings.TrimSpace(v)
	}
	if inst.ByteOrderMark != nil {
		b, err := ec.evalBoolSerializationAVT(ctx, inst.ByteOrderMark, paramByteOrderMark)
		if err != nil {
			return nil, err
		}
		base.ByteOrderMark = b
	}
	if inst.CDATASectionElements != nil {
		v, err := evalAVT(inst.CDATASectionElements)
		if err != nil {
			return nil, err
		}
		if v = strings.TrimSpace(v); v != "" {
			// Union with base cdata-section-elements from xsl:output.
			existing := make(map[string]struct{}, len(base.CDATASections))
			for _, name := range base.CDATASections {
				existing[name] = struct{}{}
			}
			for name := range strings.FieldsSeq(v) {
				resolved := resolveQName(name, inst.NSBindings)
				if _, ok := existing[resolved]; !ok {
					base.CDATASections = append(base.CDATASections, resolved)
				}
			}
		}
	}
	if inst.MediaType != nil {
		v, err := evalAVT(inst.MediaType)
		if err != nil {
			return nil, err
		}
		base.MediaType = strings.TrimSpace(v)
	}
	if inst.HTMLVersion != nil {
		v, err := evalAVT(inst.HTMLVersion)
		if err != nil {
			return nil, err
		}
		base.HTMLVersion = strings.TrimSpace(v)
	}
	if inst.IncludeContentType != nil {
		b, err := ec.evalBoolSerializationAVT(ctx, inst.IncludeContentType, paramIncludeContentType)
		if err != nil {
			return nil, err
		}
		base.IncludeContentType = &b
	}
	if inst.AllowDuplicateNames != nil {
		b, err := ec.evalBoolSerializationAVT(ctx, inst.AllowDuplicateNames, paramAllowDuplicateNames)
		if err != nil {
			return nil, err
		}
		base.AllowDuplicateNames = b
	}
	if inst.UndeclarePrefixes != nil {
		b, err := ec.evalBoolSerializationAVT(ctx, inst.UndeclarePrefixes, paramUndeclarePrefixes)
		if err != nil {
			return nil, err
		}
		base.UndeclarePrefixes = b
	}
	if inst.EscapeURIAttributes != nil {
		b, err := ec.evalBoolSerializationAVT(ctx, inst.EscapeURIAttributes, paramEscapeURIAttributes)
		if err != nil {
			return nil, err
		}
		base.EscapeURIAttributes = &b
	}
	if inst.JSONNodeOutputMethodAVT != nil {
		v, err := evalAVT(inst.JSONNodeOutputMethodAVT)
		if err != nil {
			return nil, err
		}
		base.JSONNodeOutputMethod = strings.TrimSpace(v)
	}
	if inst.NormalizationForm != nil {
		v, err := evalAVT(inst.NormalizationForm)
		if err != nil {
			return nil, err
		}
		base.NormalizationForm = strings.ToUpper(strings.TrimSpace(v))
	}
	if len(inst.SuppressIndentation) > 0 {
		// Copy the slice so a derived/handler-delivered OutputDef cannot mutate
		// the compiled instruction's SuppressIndentation backing array.
		base.SuppressIndentation = append([]string(nil), inst.SuppressIndentation...)
	}
	if inst.ItemSeparatorSet {
		if inst.ItemSeparator != nil {
			sepVal, err := inst.ItemSeparator.evaluate(ctx, ec.contextNode)
			if err != nil {
				return nil, err
			}
			base.ItemSeparator = &sepVal
		} else {
			base.ItemSeparator = nil
			base.ItemSeparatorAbsent = true
		}
	}
	if inst.BuildTree != nil {
		b, err := ec.evalBoolSerializationAVT(ctx, inst.BuildTree, paramBuildTree)
		if err != nil {
			return nil, err
		}
		base.BuildTree = &b
	}
	return &base, nil
}

// buildEffectiveOutputDef builds the effective output definition for a secondary
// result document, combining the named format with result-document overrides.
func (ec *execContext) buildEffectiveOutputDef(ctx context.Context, inst *resultDocumentInst, formatName, method string) (*OutputDef, error) {
	// Build the base in priority order (low -> high): named format (or the
	// unnamed default xsl:output when no format), then the parameter-document.
	// This mirrors evalResultDocOutputDef's default-folding; without the default
	// fold a secondary result-document with no local serialization attributes
	// would not inherit stylesheet defaults (e.g. method="json"
	// allow-duplicate-names="yes"), and the SERE0022 dup-key check below would see
	// a hard-false allow-duplicate-names and wrongly reject duplicate JSON keys
	// the default output permits. When evalResultDocOutputDef returns a non-nil
	// overrides def (hasAny or a named format), it already folded the default and
	// the parameter-document in and base is replaced below; this construction only
	// matters when overrides is nil (no serialization params and no
	// parameter-document, hence paramDocOD is nil here too).
	var base OutputDef
	paramDocOD := ec.getParamDocOutputDef(inst)
	if formatName != "" {
		if fmtDef, ok := ec.effectiveOutputs()[formatName]; ok {
			base = *cloneOutputDef(fmtDef)
		}
	} else if defDef, ok := ec.effectiveOutputs()[""]; ok {
		base = *cloneOutputDef(defDef)
	}
	if paramDocOD != nil {
		foldParamDocOverrides(&base, paramDocOD, ec.getParamDocPresence(inst))
	}
	if base.Method == "" && method != "" {
		base.Method = method
		base.MethodExplicit = true
	}
	// Apply overrides from xsl:result-document.
	// evalResultDocOutputDef already starts from the named format base and
	// applies all xsl:result-document attribute overrides, so use it as the
	// complete effective output def when available.
	overrides, err := ec.evalResultDocOutputDef(ctx, inst)
	if err != nil {
		return nil, err
	}
	if overrides != nil {
		base = *cloneOutputDef(overrides)
	}
	// Resolve character maps from the format and instruction.
	var allMaps []string
	if formatName != "" {
		if fmtDef, ok := ec.effectiveOutputs()[formatName]; ok {
			allMaps = append(allMaps, fmtDef.UseCharacterMaps...)
		}
	}
	allMaps = append(allMaps, inst.UseCharacterMaps...)
	if len(allMaps) > 0 {
		base.ResolvedCharMap = resolveCharacterMaps(ec.effectiveStylesheet(), allMaps)
	}
	return &base, nil
}

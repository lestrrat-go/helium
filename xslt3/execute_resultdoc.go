package xslt3

import (
	"context"
	"errors"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/internal/sequence"
)

// getParamDocOutputDef returns the effective parameter-document OutputDef for
// a result-document instruction, checking the per-invocation cache on
// execContext first, then falling back to the compiled instruction's field.
func (ec *execContext) getParamDocOutputDef(inst *resultDocumentInst) *OutputDef {
	if od, ok := ec.paramDocOutputDefs[inst]; ok {
		return od
	}
	return inst.ParameterDocOutputDef
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
				if err := ec.validateAndNormalizeElementContent(root, inst.TypeName); err != nil {
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
			for node, typeName := range vr.Annotations {
				ec.annotateNode(node, typeName)
			}
			for elem := range vr.NilledElements {
				ec.markNilled(elem)
			}
		}
	}

	// When the temporary document has no children (e.g. empty TVT body),
	// the xsl:document node still acts as a text-node boundary that breaks
	// the atomic adjacency chain (XSLT 3.0 §5.7.2).
	out := ec.currentOutput()
	if tmpDoc.FirstChild() == nil {
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
				helium.UnlinkNode(child)
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
			return inst.Format, nil
		}
		v = strings.TrimSpace(v)
		if v != "" && !strings.HasPrefix(v, "Q{") {
			// XTDE0290: prefix must have a namespace binding
			if idx := strings.IndexByte(v, ':'); idx >= 0 {
				prefix := v[:idx]
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
		if outDef, ok := ec.stylesheet.outputs[format]; ok {
			return outDef.Method
		}
	}
	// Default output definition.
	if outDef, ok := ec.stylesheet.outputs[""]; ok {
		return outDef.Method
	}
	return methodXML
}

// isItemSerializationMethod returns true when the output method supports
// non-node items (maps, arrays, function items) without XTDE0450.
func isItemSerializationMethod(method string) bool {
	return method == methodJSON || method == methodAdaptive
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

	// Check for duplicate URI (XTDE1490).
	if _, used := ec.usedResultURIs[href]; used {
		return dynamicError(errCodeXTDE1490, "two result documents written to same URI: %q", href)
	}

	isPrimary := href == ""

	if isPrimary && ec.primaryClaimedImplicitly {
		return dynamicError(errCodeXTRE1495, "primary output URI already has implicit content")
	}

	ec.usedResultURIs[href] = struct{}{}

	// Resolve the effective format name (static or avt).
	effectiveFormat, fmtErr := ec.resolveResultDocFormat(ctx, inst)
	if fmtErr != nil {
		return fmtErr
	}

	// XTDE1460: the format attribute must reference a declared xsl:output.
	if effectiveFormat != "" {
		if _, ok := ec.stylesheet.outputs[effectiveFormat]; !ok {
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
			if pdErr == nil && pdHref != "" {
				outDef := &OutputDef{}
				baseURI := ec.effectiveStaticBaseURI()
				if loadErr := loadParameterDocumentFromFile(ctx, outDef, baseURI, pdHref); loadErr == nil {
					if ec.paramDocOutputDefs == nil {
						ec.paramDocOutputDefs = make(map[*resultDocumentInst]*OutputDef)
					}
					ec.paramDocOutputDefs[inst] = outDef
				}
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
		if outDef, ok := ec.stylesheet.outputs[effectiveFormat]; ok && outDef.ItemSeparator != nil {
			itemSep = outDef.ItemSeparator
		}
	} else if outDef, ok := ec.stylesheet.outputs[""]; ok && outDef.ItemSeparator != nil {
		itemSep = outDef.ItemSeparator
	}

	// Track the current output URI for current-output-uri().
	// For secondary outputs, resolve relative href against the current output URI.
	savedOutputURI := ec.currentOutputURI
	if href != "" && savedOutputURI != "" {
		resolved := helium.BuildURI(href, savedOutputURI)
		if resolved != "" {
			ec.currentOutputURI = resolved
		} else {
			ec.currentOutputURI = href
		}
	} else if href != "" {
		ec.currentOutputURI = href
	}
	// For primary output (href==""), currentOutputURI stays unchanged.
	defer func() { ec.currentOutputURI = savedOutputURI }()

	if isPrimary {
		v := inst.Validation
		if inst.TypeName != "" && v == "" {
			// type attribute without explicit validation: build into temp doc, validate type, copy.
			tmpDoc := helium.NewDefaultDocument()
			ec.outputStack = append(ec.outputStack, &outputFrame{doc: tmpDoc, current: tmpDoc, itemSeparator: itemSep})
			if err := ec.executeSequenceConstructor(ctx, inst.Body); err != nil {
				ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
				return err
			}
			ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
			root := findDocumentElement(tmpDoc)
			if root != nil && ec.schemaRegistry != nil {
				if err := ec.validateAndNormalizeElementContent(root, inst.TypeName); err != nil {
					if xsltErr, ok := errors.AsType[*XSLTError](err); ok && xsltErr.Code == errCodeXTTE1510 {
						return dynamicError(errCodeXTTE1540,
							"result document content does not match declared type %s: %v", inst.TypeName, xsltErr.Message)
					}
					return err
				}
			}
			primaryFrame := ec.outputStack[0]
			for child := tmpDoc.FirstChild(); child != nil; child = child.NextSibling() {
				if err := primaryFrame.doc.AddChild(child); err != nil {
					return err
				}
			}
			return nil
		}
		if v == validationStrict || v == validationLax {
			// When validation is requested for the primary output, build into a
			// temporary document, validate it, then copy children to the primary
			// output. This is the only way we can inspect the complete document
			// structure before emitting it.
			tmpDoc := helium.NewDefaultDocument()
			ec.outputStack = append(ec.outputStack, &outputFrame{doc: tmpDoc, current: tmpDoc, itemSeparator: itemSep})
			if err := ec.executeSequenceConstructor(ctx, inst.Body); err != nil {
				ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
				return err
			}
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
				for node, typeName := range vr.Annotations {
					ec.annotateNode(node, typeName)
				}
				for elem := range vr.NilledElements {
					ec.markNilled(elem)
				}
			}
			// Copy validated children into the primary output.
			primaryFrame := ec.outputStack[0]
			for child := tmpDoc.FirstChild(); child != nil; child = child.NextSibling() {
				if err := primaryFrame.doc.AddChild(child); err != nil {
					return err
				}
			}
			return nil
		}
		effectiveMethod := ec.resolveResultDocMethod(ctx, inst)
		buildTreeNo := inst.BuildTree != nil && !*inst.BuildTree

		// When build-tree="no", execute into a temporary document,
		// then extract children and pending items as a raw sequence
		// for serialization with item-separator.
		if buildTreeNo && isItemSerializationMethod(effectiveMethod) {
			tmpDoc := helium.NewDefaultDocument()
			tmpRoot, tmpErr := tmpDoc.CreateElement("_tmp")
			if tmpErr != nil {
				return tmpErr
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
			out.pendingItems = append(out.pendingItems, frame.pendingItems...)

			if overrides, err := ec.evalResultDocOutputDef(ctx, inst); err != nil {
				return err
			} else if overrides != nil {
				ec.primaryOutputOverrides = overrides
			}
			return nil
		}

		// Write directly to the primary output (base frame).
		savedStack := ec.outputStack
		ec.outputStack = ec.outputStack[:1] // keep only the base frame
		ec.insideResultDocPrimary = true
		savedSep := ec.outputStack[0].itemSeparator
		ec.outputStack[0].itemSeparator = itemSep
		savedMethod := ec.currentResultDocMethod
		ec.currentResultDocMethod = effectiveMethod
		if err := ec.executeSequenceConstructor(ctx, inst.Body); err != nil {
			ec.insideResultDocPrimary = false
			ec.currentResultDocMethod = savedMethod
			ec.outputStack[0].itemSeparator = savedSep
			ec.outputStack = savedStack
			return err
		}
		// Validate JSON duplicate keys (SERE0022) when allow-duplicate-names is not "yes".
		if effectiveMethod == methodJSON {
			allowDupes := false // default: allow-duplicate-names=no per XSLT 3.0 §20
			if inst.AllowDuplicateNames != nil {
				adnVal, adnErr := inst.AllowDuplicateNames.evaluate(ctx, ec.contextNode)
				if adnErr == nil {
					adnVal = strings.TrimSpace(adnVal)
					if adnVal == lexicon.ValueYes || adnVal == "true" || adnVal == "1" {
						allowDupes = true
					}
				}
			}
			if !allowDupes {
				out := ec.outputStack[0]
				if err := validateJSONItems(out.pendingItems); err != nil {
					ec.insideResultDocPrimary = false
					ec.currentResultDocMethod = savedMethod
					ec.outputStack[0].itemSeparator = savedSep
					ec.outputStack = savedStack
					return err
				}
			}
		}
		ec.insideResultDocPrimary = false
		ec.currentResultDocMethod = savedMethod
		ec.outputStack[0].itemSeparator = savedSep
		ec.outputStack = savedStack
		// Propagate character map names to the primary output frame.
		// Include maps from the named format (xsl:output) first, then
		// maps from xsl:result-document itself (higher priority).
		var allMaps []string
		if effectiveFormat != "" {
			if fmtDef, ok := ec.stylesheet.outputs[effectiveFormat]; ok {
				allMaps = append(allMaps, fmtDef.UseCharacterMaps...)
				// Also propagate resolved character maps from parameter-document.
				if len(fmtDef.ResolvedCharMap) > 0 {
					ec.primaryResolvedCharMap = fmtDef.ResolvedCharMap
				}
			}
		}
		allMaps = append(allMaps, inst.UseCharacterMaps...)
		if len(allMaps) > 0 {
			ec.primaryCharacterMaps = allMaps
		}
		// Capture serialization parameter overrides from xsl:result-document.
		if overrides, err := ec.evalResultDocOutputDef(ctx, inst); err != nil {
			return err
		} else if overrides != nil {
			ec.primaryOutputOverrides = overrides
		}
		return nil
	}

	// Secondary output: execute body into a temporary document.
	tmpDoc := helium.NewDefaultDocument()

	// Set the document URL so that base-uri() returns the correct value.
	// Resolve relative href against the stylesheet base URI.
	resolvedHref := href
	if ec.stylesheet.baseURI != "" {
		resolved := helium.BuildURI(href, ec.stylesheet.baseURI)
		if resolved != "" {
			resolvedHref = resolved
		}
	}
	tmpDoc.SetURL(resolvedHref)

	effectiveMethod := ec.resolveResultDocMethod(ctx, inst)
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

	// For json/adaptive serialization, store captured items.
	if isItemSerializationMethod(effectiveMethod) && len(frame.pendingItems) > 0 {
		ec.resultDocItems[href] = frame.pendingItems
	}

	// Always store the effective output definition for secondary result documents
	// so that serialization parameters (omit-xml-declaration, indent, etc.) from
	// the named format are applied when serializing the result.
	effOutDef, err := ec.buildEffectiveOutputDef(ctx, inst, effectiveFormat, effectiveMethod)
	if err == nil && effOutDef != nil {
		ec.resultDocOutputDefs[href] = effOutDef
	}

	// Apply type validation for secondary result documents (type="...").
	if inst.TypeName != "" && inst.Validation == "" {
		root := findDocumentElement(tmpDoc)
		if root != nil && ec.schemaRegistry != nil {
			if err := ec.validateAndNormalizeElementContent(root, inst.TypeName); err != nil {
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
			for node, typeName := range vr.Annotations {
				ec.annotateNode(node, typeName)
			}
			for elem := range vr.NilledElements {
				ec.markNilled(elem)
			}
		}
	} else if inst.Validation == validationStrip {
		root := findDocumentElement(tmpDoc)
		if root != nil {
			ec.stripAnnotations(root)
		}
	}

	// Store the secondary result document.
	ec.resultDocuments[href] = tmpDoc
	return nil
}

// evalResultDocOutputDef evaluates serialization parameter AVTs on
// xsl:result-document and returns an OutputDef with the overrides.
// Returns nil if no serialization parameters are specified.
func (ec *execContext) evalResultDocOutputDef(ctx context.Context, inst *resultDocumentInst) (*OutputDef, error) {
	hasAny := inst.MethodAVT != nil || inst.Standalone != nil || inst.Indent != nil ||
		inst.OmitXMLDeclaration != nil || inst.DoctypeSystem != nil || inst.DoctypePublic != nil ||
		inst.CDATASectionElements != nil || inst.Encoding != nil || inst.OutputVersion != nil ||
		inst.ByteOrderMark != nil || inst.EscapeURIAttributes != nil ||
		inst.JSONNodeOutputMethodAVT != nil || inst.NormalizationForm != nil ||
		ec.getParamDocOutputDef(inst) != nil ||
		inst.ItemSeparatorSet || inst.BuildTree != nil
	effectiveFormat, fmtErr := ec.resolveResultDocFormat(ctx, inst)
	if fmtErr != nil {
		return nil, fmtErr
	}
	if !hasAny && effectiveFormat == "" {
		return nil, nil
	}

	// Start with parameter-document defaults (lowest priority).
	var base OutputDef
	paramDocOD := ec.getParamDocOutputDef(inst)
	if paramDocOD != nil {
		base = *paramDocOD
	}
	// Named format overrides parameter-document.
	if effectiveFormat != "" {
		if fmtDef, ok := ec.stylesheet.outputs[effectiveFormat]; ok {
			base = *fmtDef
		}
	} else if paramDocOD == nil {
		if defDef, ok := ec.stylesheet.outputs[""]; ok {
			base = *defDef
		}
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
		case "true", "1":
			v = lexicon.ValueYes
		case "false", "0":
			v = lexicon.ValueNo
		default:
			v = strings.TrimSpace(v)
		}
		base.Standalone = v
	}
	if inst.Indent != nil {
		v, err := evalAVT(inst.Indent)
		if err != nil {
			return nil, err
		}
		b, _ := parseXSDBool(strings.TrimSpace(v))
		base.Indent = b
	}
	if inst.OmitXMLDeclaration != nil {
		v, err := evalAVT(inst.OmitXMLDeclaration)
		if err != nil {
			return nil, err
		}
		b, _ := parseXSDBool(strings.TrimSpace(v))
		base.OmitDeclaration = b
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
	if inst.ByteOrderMark != nil {
		v, err := evalAVT(inst.ByteOrderMark)
		if err != nil {
			return nil, err
		}
		if b, ok := parseXSDBool(strings.TrimSpace(v)); ok {
			base.ByteOrderMark = b
		}
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
			for _, name := range strings.Fields(v) {
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
		v, err := evalAVT(inst.IncludeContentType)
		if err != nil {
			return nil, err
		}
		b, _ := parseXSDBool(strings.TrimSpace(v))
		base.IncludeContentType = &b
	}
	if inst.EscapeURIAttributes != nil {
		v, err := evalAVT(inst.EscapeURIAttributes)
		if err != nil {
			return nil, err
		}
		if b, ok := parseXSDBool(strings.TrimSpace(v)); ok {
			base.EscapeURIAttributes = &b
		}
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
		base.SuppressIndentation = inst.SuppressIndentation
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
		base.BuildTree = inst.BuildTree
	}
	return &base, nil
}

// buildEffectiveOutputDef builds the effective output definition for a secondary
// result document, combining the named format with result-document overrides.
func (ec *execContext) buildEffectiveOutputDef(ctx context.Context, inst *resultDocumentInst, formatName, method string) (*OutputDef, error) {
	var base OutputDef
	// Start with parameter-document defaults (lowest priority).
	if pd := ec.getParamDocOutputDef(inst); pd != nil {
		base = *pd
	}
	// Named format overrides parameter-document.
	if formatName != "" {
		if fmtDef, ok := ec.stylesheet.outputs[formatName]; ok {
			base = *fmtDef
		}
	}
	if base.Method == "" && method != "" {
		base.Method = method
		base.MethodExplicit = true
	}
	// Apply overrides from xsl:result-document
	overrides, err := ec.evalResultDocOutputDef(ctx, inst)
	if err != nil {
		return nil, err
	}
	if overrides != nil {
		if overrides.Method != "" {
			base.Method = overrides.Method
			base.MethodExplicit = true
		}
		if overrides.Encoding != "" {
			base.Encoding = overrides.Encoding
		}
		if overrides.JSONNodeOutputMethod != "" {
			base.JSONNodeOutputMethod = overrides.JSONNodeOutputMethod
		}
		if len(overrides.ResolvedCharMap) > 0 && base.ResolvedCharMap == nil {
			base.ResolvedCharMap = overrides.ResolvedCharMap
		}
	}
	// Resolve character maps from the format and instruction.
	var allMaps []string
	if formatName != "" {
		if fmtDef, ok := ec.stylesheet.outputs[formatName]; ok {
			allMaps = append(allMaps, fmtDef.UseCharacterMaps...)
		}
	}
	allMaps = append(allMaps, inst.UseCharacterMaps...)
	if len(allMaps) > 0 {
		base.ResolvedCharMap = resolveCharacterMaps(ec.stylesheet, allMaps)
	}
	return &base, nil
}

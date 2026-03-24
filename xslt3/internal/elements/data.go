package elements

import "github.com/lestrrat-go/helium/internal/lexicon"

func elementDefs() map[string]ElementInfo {
	return map[string]ElementInfo{
		// ── Root elements ──────────────────────────────────────────────
		lexicon.XSLTElementStylesheet: {
			MinVersion: "1.0", Context: CtxRoot, Implemented: true,
		},
		lexicon.XSLTElementTransform: {
			MinVersion: "1.0", Context: CtxRoot, Implemented: true,
		},
		lexicon.XSLTElementPackage: {
			MinVersion: "3.0", Context: CtxRoot, Implemented: true,
		},

		// ── Top-level declarations ─────────────────────────────────────
		lexicon.XSLTElementImport: {
			MinVersion: "1.0", Context: CtxTopLevel, Implemented: true,
			AllowedAttrs: attrSet("href", "use-when"),
		},
		lexicon.XSLTElementInclude: {
			MinVersion: "1.0", Context: CtxTopLevel, Implemented: true,
			AllowedAttrs: attrSet("href", "use-when"),
		},
		lexicon.XSLTElementTemplate: {
			MinVersion: "1.0", Context: CtxTopLevel, Implemented: true,
			AllowedAttrs: attrSet("match", "name", "priority", "mode", "as", "visibility", "use-when"),
		},
		lexicon.XSLTElementVariable: {
			MinVersion: "1.0", Context: CtxTopLevel | CtxInstruction, Implemented: true,
			AllowedAttrs: attrSet("name", "select", "as", "static", "visibility"),
		},
		lexicon.XSLTElementParam: {
			MinVersion: "1.0", Context: CtxTopLevel | CtxChildOnly, Implemented: true,
			AllowedAttrs: attrSet("name", "select", "as", "required", "tunnel", "static"),
			Parents:      []string{"template", "function", "iterate"},
		},
		lexicon.XSLTElementKey: {
			MinVersion: "1.0", Context: CtxTopLevel, Implemented: true,
			AllowedAttrs: attrSet("name", "match", "use", "collation", "composite", "use-when"),
		},
		lexicon.XSLTElementOutput: {
			MinVersion: "1.0", Context: CtxTopLevel, Implemented: true,
			AllowedAttrs: attrSet(
				"name", "method", "version", "encoding",
				"omit-xml-declaration", "standalone", "doctype-public",
				"doctype-system", "cdata-section-elements", "indent",
				"media-type", "byte-order-mark", "escape-uri-attributes",
				"include-content-type", "normalization-form",
				"undeclare-prefixes", "use-character-maps",
				"suppress-indentation", "html-version",
				"item-separator", "json-node-output-method",
				"parameter-document", "build-tree",
				"allow-duplicate-names", "use-when",
			),
		},
		lexicon.XSLTElementStripSpace: {
			MinVersion: "1.0", Context: CtxTopLevel, Implemented: true,
			AllowedAttrs: attrSet("elements"),
		},
		lexicon.XSLTElementPreserveSpace: {
			MinVersion: "1.0", Context: CtxTopLevel, Implemented: true,
			AllowedAttrs: attrSet("elements"),
		},
		lexicon.XSLTElementFunction: {
			MinVersion: "2.0", Context: CtxTopLevel, Implemented: true,
			AllowedAttrs: attrSet(
				"name", "as", "visibility", "streamable",
				"streamability", "override-extension-function", "override",
				"identity-sensitive", "cache", "new-each-time", "use-when",
			),
		},
		lexicon.XSLTElementDecimalFormat: {
			MinVersion: "1.0", Context: CtxTopLevel, Implemented: true,
			AllowedAttrs: attrSet(
				"name", "decimal-separator", "grouping-separator",
				"infinity", "minus-sign", "NaN", "percent",
				"per-mille", "zero-digit", "digit",
				"pattern-separator", "exponent-separator", "use-when",
			),
		},
		lexicon.XSLTElementMode: {
			MinVersion: "3.0", Context: CtxTopLevel, Implemented: true,
			AllowedAttrs: attrSet(
				"name", "streamable", "on-no-match", "on-multiple-match",
				"warning-on-no-match", "warning-on-multiple-match",
				"typed", "visibility", "use-when", "use-accumulators",
			),
		},
		lexicon.XSLTElementImportSchema: {
			MinVersion: "2.0", Context: CtxTopLevel, Implemented: true,
			AllowedAttrs: attrSet("namespace", "schema-location", "use-when"),
		},
		lexicon.XSLTElementAccumulator: {
			MinVersion: "3.0", Context: CtxTopLevel, Implemented: true,
			AllowedAttrs: attrSet("name", "as", "initial-value", "streamable", "use-when"),
		},
		lexicon.XSLTElementAttributeSet: {
			MinVersion: "1.0", Context: CtxTopLevel, Implemented: true,
			AllowedAttrs: attrSet("name", "use-attribute-sets", "visibility", "streamable", "use-when"),
		},
		lexicon.XSLTElementCharacterMap: {
			MinVersion: "2.0", Context: CtxTopLevel, Implemented: true,
			AllowedAttrs: attrSet("name", "use-character-maps", "use-when"),
		},
		lexicon.XSLTElementNamespaceAlias: {
			MinVersion: "1.0", Context: CtxTopLevel, Implemented: true,
			AllowedAttrs: attrSet("stylesheet-prefix", "result-prefix", "use-when"),
		},
		lexicon.XSLTElementExpose: {
			MinVersion: "3.0", Context: CtxTopLevel, Implemented: true,
			AllowedAttrs: attrSet("component", "names", "visibility"),
		},
		lexicon.XSLTElementGlobalContextItem: {
			MinVersion: "3.0", Context: CtxTopLevel, Implemented: true,
			AllowedAttrs: attrSet("as", "use", "use-when"),
		},
		lexicon.XSLTElementUsePackage: {
			MinVersion: "3.0", Context: CtxTopLevel, Implemented: true,
			AllowedAttrs: attrSet("name", "package-version", "use-when"),
		},

		// ── Instruction elements (sequence constructors) ───────────────
		lexicon.XSLTElementApplyTemplates: {
			MinVersion: "1.0", Context: CtxInstruction, Implemented: true,
			AllowedAttrs: attrSet("select", "mode"),
		},
		lexicon.XSLTElementCallTemplate: {
			MinVersion: "1.0", Context: CtxInstruction, Implemented: true,
			AllowedAttrs: attrSet("name"),
		},
		lexicon.XSLTElementValueOf: {
			MinVersion: "1.0", Context: CtxInstruction, Implemented: true,
			AllowedAttrs: attrSet("select", "separator", "disable-output-escaping"),
		},
		lexicon.XSLTElementText: {
			MinVersion: "1.0", Context: CtxInstruction, Implemented: true,
			AllowedAttrs: attrSet("disable-output-escaping"),
		},
		lexicon.XSLTElementElement: {
			MinVersion: "1.0", Context: CtxInstruction, Implemented: true,
			AllowedAttrs: attrSet("name", "namespace", "inherit-namespaces", "use-attribute-sets", "type", "validation"),
		},
		lexicon.XSLTElementAttribute: {
			MinVersion: "1.0", Context: CtxInstruction, Implemented: true,
			AllowedAttrs: attrSet("name", "namespace", "select", "separator", "type", "validation"),
		},
		lexicon.XSLTElementComment: {
			MinVersion: "1.0", Context: CtxInstruction, Implemented: true,
			AllowedAttrs: attrSet("select"),
		},
		lexicon.XSLTElementProcessingInstruction: {
			MinVersion: "1.0", Context: CtxInstruction, Implemented: true,
			AllowedAttrs: attrSet("name", "select"),
		},
		lexicon.XSLTElementIf: {
			MinVersion: "1.0", Context: CtxInstruction, Implemented: true,
			AllowedAttrs: attrSet("test"),
		},
		lexicon.XSLTElementChoose: {
			MinVersion: "1.0", Context: CtxInstruction, Implemented: true,
		},
		lexicon.XSLTElementForEach: {
			MinVersion: "1.0", Context: CtxInstruction, Implemented: true,
			AllowedAttrs: attrSet("select"),
		},
		lexicon.XSLTElementCopy: {
			MinVersion: "1.0", Context: CtxInstruction, Implemented: true,
			AllowedAttrs: attrSet("select", "copy-namespaces", "inherit-namespaces", "use-attribute-sets", "type", "validation"),
		},
		lexicon.XSLTElementCopyOf: {
			MinVersion: "1.0", Context: CtxInstruction, Implemented: true,
			AllowedAttrs: attrSet("select", "copy-namespaces", "copy-accumulators", "type", "validation"),
		},
		lexicon.XSLTElementNumber: {
			MinVersion: "1.0", Context: CtxInstruction, Implemented: true,
			AllowedAttrs: attrSet("value", "select", "level", "count", "from", "format", "lang", "letter-value", "ordinal", "grouping-separator", "grouping-size"),
		},
		lexicon.XSLTElementMessage: {
			MinVersion: "1.0", Context: CtxInstruction, Implemented: true,
			AllowedAttrs: attrSet("select", "terminate", "error-code"),
		},
		lexicon.XSLTElementNamespace: {
			MinVersion: "2.0", Context: CtxInstruction, Implemented: true,
			AllowedAttrs: attrSet("name", "select"),
		},
		lexicon.XSLTElementSequence: {
			MinVersion: "2.0", Context: CtxInstruction, Implemented: true,
			AllowedAttrs: attrSet("select", "as"),
		},
		lexicon.XSLTElementPerformSort: {
			MinVersion: "2.0", Context: CtxInstruction, Implemented: true,
			AllowedAttrs: attrSet("select"),
		},
		lexicon.XSLTElementNextMatch: {
			MinVersion: "2.0", Context: CtxInstruction, Implemented: true,
		},
		lexicon.XSLTElementApplyImports: {
			MinVersion: "1.0", Context: CtxInstruction, Implemented: true,
		},
		lexicon.XSLTElementDocument: {
			MinVersion: "2.0", Context: CtxInstruction, Implemented: true,
			AllowedAttrs: attrSet("type", "validation"),
		},
		lexicon.XSLTElementResultDocument: {
			MinVersion: "2.0", Context: CtxInstruction, Implemented: true,
			AllowedAttrs: attrSet(
				"format", "href", "method", "version", "encoding",
				"omit-xml-declaration", "standalone", "doctype-public",
				"doctype-system", "cdata-section-elements", "indent",
				"media-type", "byte-order-mark", "escape-uri-attributes",
				"include-content-type", "normalization-form",
				"undeclare-prefixes", "use-character-maps",
				"output-version", "type", "validation",
			),
		},
		lexicon.XSLTElementWherePopulated: {
			MinVersion: "3.0", Context: CtxInstruction, Implemented: true,
		},
		lexicon.XSLTElementOnEmpty: {
			MinVersion: "3.0", Context: CtxInstruction, Implemented: true,
		},
		lexicon.XSLTElementOnNonEmpty: {
			MinVersion: "3.0", Context: CtxInstruction, Implemented: true,
		},
		lexicon.XSLTElementTry: {
			MinVersion: "3.0", Context: CtxInstruction, Implemented: true,
			AllowedAttrs: attrSet("select", "rollback-output"),
		},
		lexicon.XSLTElementForEachGroup: {
			MinVersion: "2.0", Context: CtxInstruction, Implemented: true,
			AllowedAttrs: attrSet("select", "group-by", "group-adjacent", "group-starting-with", "group-ending-with", "composite", "collation"),
		},
		lexicon.XSLTElementMap: {
			MinVersion: "3.0", Context: CtxInstruction, Implemented: true,
		},
		lexicon.XSLTElementMapEntry: {
			MinVersion: "3.0", Context: CtxInstruction, Implemented: true,
			AllowedAttrs: attrSet("key", "select"),
		},
		lexicon.XSLTElementAssert: {
			MinVersion: "3.0", Context: CtxInstruction, Implemented: true,
			AllowedAttrs: attrSet("test", "select", "error-code"),
		},
		lexicon.XSLTElementAnalyzeString: {
			MinVersion: "2.0", Context: CtxInstruction, Implemented: true,
			AllowedAttrs: attrSet("select", "regex", "flags"),
		},
		lexicon.XSLTElementEvaluate: {
			MinVersion: "3.0", Context: CtxInstruction, Implemented: true,
			AllowedAttrs: attrSet("xpath", "as", "base-uri", "namespace-context", "schema-aware"),
		},
		lexicon.XSLTElementSourceDocument: {
			MinVersion: "3.0", Context: CtxInstruction, Implemented: true,
			AllowedAttrs: attrSet("href", "streamable", "use-accumulators", "type", "validation"),
		},
		lexicon.XSLTElementIterate: {
			MinVersion: "3.0", Context: CtxInstruction, Implemented: true,
			AllowedAttrs: attrSet("select"),
		},
		lexicon.XSLTElementFork: {
			MinVersion: "3.0", Context: CtxInstruction, Implemented: true,
		},
		lexicon.XSLTElementMerge: {
			MinVersion: "3.0", Context: CtxInstruction, Implemented: true,
		},

		// ── Child-only elements ────────────────────────────────────────
		lexicon.XSLTElementSort: {
			MinVersion: "1.0", Context: CtxChildOnly, Implemented: true,
			AllowedAttrs: attrSet("select", "order", "data-type", "case-order", "lang", "collation", "stable"),
			Parents:      []string{"apply-templates", "for-each", "for-each-group", "perform-sort"},
		},
		lexicon.XSLTElementWhen: {
			MinVersion: "1.0", Context: CtxChildOnly, Implemented: true,
			AllowedAttrs: attrSet("test"),
			Parents:      []string{"choose"},
		},
		lexicon.XSLTElementOtherwise: {
			MinVersion: "1.0", Context: CtxChildOnly, Implemented: true,
			Parents: []string{"choose"},
		},
		lexicon.XSLTElementCatch: {
			MinVersion: "3.0", Context: CtxChildOnly, Implemented: true,
			AllowedAttrs: attrSet("errors", "select"),
			Parents:      []string{"try"},
		},
		lexicon.XSLTElementWithParam: {
			MinVersion: "1.0", Context: CtxChildOnly, Implemented: true,
			AllowedAttrs: attrSet("name", "select", "as", "tunnel"),
			Parents:      []string{"apply-templates", "call-template", "next-match", "apply-imports", "evaluate"},
		},
		lexicon.XSLTElementFallback: {
			MinVersion: "1.0", Context: CtxChildOnly, Implemented: true,
			// Parents is nil: fallback can appear inside any XSLT instruction
			// as a forwards-compatibility mechanism; validated at compile time.
		},
		lexicon.XSLTElementMatchingSubstring: {
			MinVersion: "2.0", Context: CtxChildOnly, Implemented: true,
			Parents: []string{"analyze-string"},
		},
		lexicon.XSLTElementNonMatchingSubstring: {
			MinVersion: "2.0", Context: CtxChildOnly, Implemented: true,
			Parents: []string{"analyze-string"},
		},
		lexicon.XSLTElementOnCompletion: {
			MinVersion: "3.0", Context: CtxChildOnly, Implemented: true,
			AllowedAttrs: attrSet("select"),
			Parents:      []string{"iterate"},
		},
		lexicon.XSLTElementMergeSource: {
			MinVersion: "3.0", Context: CtxChildOnly, Implemented: true,
			AllowedAttrs: attrSet("name", "for-each-item", "for-each-source", "select", "streamable", "sort-before-merge", "use-accumulators", "validation"),
			Parents:      []string{"merge"},
		},
		lexicon.XSLTElementMergeAction: {
			MinVersion: "3.0", Context: CtxChildOnly, Implemented: true,
			Parents: []string{"merge"},
		},
		lexicon.XSLTElementMergeKey: {
			MinVersion: "3.0", Context: CtxChildOnly, Implemented: true,
			AllowedAttrs: attrSet("select", "order", "collation", "lang", "data-type", "case-order"),
			Parents:      []string{"merge-source"},
		},
		lexicon.XSLTElementAccumulatorRule: {
			MinVersion: "3.0", Context: CtxChildOnly, Implemented: true,
			AllowedAttrs: attrSet("match", "phase", "select", "new-value"),
			Parents:      []string{"accumulator"},
		},
		lexicon.XSLTElementOutputCharacter: {
			MinVersion: "2.0", Context: CtxChildOnly, Implemented: true,
			AllowedAttrs: attrSet("character", "string"),
			Parents:      []string{"character-map"},
		},
		lexicon.XSLTElementContextItem: {
			MinVersion: "3.0", Context: CtxChildOnly, Implemented: true,
			AllowedAttrs: attrSet("as", "use"),
			Parents:      []string{"template"},
		},
		lexicon.XSLTElementBreak: {
			MinVersion: "3.0", Context: CtxChildOnly, Implemented: true,
			AllowedAttrs: attrSet("select"),
			Parents:      []string{"iterate"},
		},
		lexicon.XSLTElementNextIteration: {
			MinVersion: "3.0", Context: CtxChildOnly, Implemented: true,
			Parents: []string{"iterate"},
		},

		// ── Recognized but not in elem* constants ──────────────────────
		lexicon.XSLTElementAccept: {
			MinVersion: "3.0", Context: CtxChildOnly, Implemented: true,
			AllowedAttrs: attrSet("component", "names", "visibility"),
			Parents:      []string{"use-package"},
		},
		lexicon.XSLTElementOverride: {
			MinVersion: "3.0", Context: CtxChildOnly, Implemented: true,
			Parents: []string{"use-package"},
		},
		lexicon.XSLTElementArray: {
			MinVersion: "3.0", Context: CtxInstruction, Implemented: true,
		},
		lexicon.XSLTElementSchema: {
			// Internal representation; not a standard XSLT instruction.
			MinVersion: "1.0", Implemented: false,
		},
	}
}

// attrSet is a helper that builds a map[string]struct{} from a list of names.
func attrSet(names ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(names))
	for _, n := range names {
		m[n] = struct{}{}
	}
	return m
}

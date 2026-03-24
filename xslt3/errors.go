package xslt3

import (
	"errors"
	"fmt"
)

// XSLT error codes per the W3C XSLT 3.0 specification.
// Sorted by prefix (FODC, SEPM, SERE, SESU, XPST, XSLT, XTDE, XTMM, XTRE, XTSE, XTTE)
// then numerically within each prefix.
const (
	// FODC — Functions and Operators: Document Collection
	errCodeFODC0002 = "FODC0002" // document() / xsl:source-document load or parse error
	errCodeFOTY0013 = "FOTY0013" // atomization of map/function items
	errCodeFOXT0001 = "FOXT0001" // fn:transform invalid argument shape
	errCodeFOXT0002 = "FOXT0002" // fn:transform missing required option or resolver
	errCodeFOXT0003 = "FOXT0003" // fn:transform stylesheet/package/serialization failure
	errCodeFOXT0004 = "FOXT0004" // fn:transform recursion depth exceeded

	// SEPM/SERE/SESU — Serialization errors
	errCodeSEPM0004 = "SEPM0004" // version/standalone constraint violation
	errCodeSEPM0009 = "SEPM0009" // omit-xml-declaration conflicts with standalone/doctype
	errCodeSEPM0010 = "SEPM0010" // undeclare-prefixes="yes" requires XML 1.1
	errCodeSEPM0016 = "SEPM0016" // invalid serialization parameter value
	errCodeSERE0012 = "SERE0012" // invalid character in text
	errCodeSERE0014 = "SERE0014" // invalid character in comment or PI
	errCodeSERE0015 = "SERE0015" // invalid character in namespace
	errCodeSERE0022 = "SERE0022" // duplicate keys in JSON map
	errCodeSESU0007 = "SESU0007" // unsupported output version
	errCodeSESU0011 = "SESU0011" // unsupported normalization form
	errCodeSESU0013 = "SESU0013" // unsupported encoding

	// XPST/XPDY/XPTY — XPath errors
	errCodeXPDY0002 = "XPDY0002" // context item is absent
	errCodeXPST0003 = "XPST0003" // invalid XPath/XSLT type expression
	errCodeXPST0008 = "XPST0008" // undeclared variable / circular static param reference
	errCodeXPST0017 = "XPST0017" // invalid function call in pattern/static context
	errCodeXPTY0004 = "XPTY0004" // type mismatch

	// XSLT — Generic
	errCodeXSLT0000 = "XSLT0000" // generic catch code for non-XSLT errors

	// XTDE — Dynamic Errors
	errCodeXTDE0030 = "XTDE0030" // invalid language code (xsl:number, xsl:sort, etc.)
	errCodeXTDE0290 = "XTDE0290" // invalid URI for xsl:result-document output
	errCodeXTDE0040 = "XTDE0040" // initial template not public
	errCodeXTDE0041 = "XTDE0041" // initial function not found or not public
	errCodeXTDE0044 = "XTDE0044" // initial mode not public in package
	errCodeXTDE0045 = "XTDE0045" // evaluation error in avt
	errCodeXTDE0050 = "XTDE0050" // required stylesheet parameter not supplied
	errCodeXTDE0060 = "XTDE0060" // required template parameter not supplied / invalid call-template target
	errCodeXTDE0160 = "XTDE0160" // multiple output documents same URI
	errCodeXTDE0410 = "XTDE0410" // duplicate parameter
	errCodeXTDE0420 = "XTDE0420" // namespace conflict
	errCodeXTDE0430 = "XTDE0430" // variable type error
	errCodeXTDE0440 = "XTDE0440" // default namespace on element in no namespace
	errCodeXTDE0450 = "XTDE0450" // non-node item in result tree
	errCodeXTDE0540 = "XTDE0540" // ambiguous rule match (on-multiple-match=fail)
	errCodeXTDE0555 = "XTDE0555" // on-no-match=fail with no matching template
	errCodeXTDE0560 = "XTDE0560" // copy-of type error
	errCodeXTDE0640 = "XTDE0640" // circular reference
	errCodeXTDE0700 = "XTDE0700" // sort key error
	errCodeXTDE0820 = "XTDE0820" // dynamic type error in template match
	errCodeXTDE0830 = "XTDE0830" // undeclared prefix in computed element name
	errCodeXTDE0835 = "XTDE0835" // xsl:message terminate=yes
	errCodeXTDE0850 = "XTDE0850" // xsl:attribute name is not a valid QName
	errCodeXTDE0855 = "XTDE0855" // xsl:attribute name evaluates to "xmlns"
	errCodeXTDE0860 = "XTDE0860" // undeclared prefix in computed attribute name
	errCodeXTDE0865 = "XTDE0865" // reserved xmlns namespace URI in xsl:attribute
	errCodeXTDE0890 = "XTDE0890" // xsl:processing-instruction name is not a valid NCName/PITarget
	errCodeXTDE0905 = "XTDE0905" // reserved xmlns namespace URI in xsl:namespace
	errCodeXTDE0920 = "XTDE0920" // xmlns prefix or invalid NCName in xsl:namespace name
	errCodeXTDE0925 = "XTDE0925" // xml namespace mismatch in xsl:namespace
	errCodeXTDE0930 = "XTDE0930" // non-empty prefix with zero-length namespace URI in xsl:namespace
	errCodeXTDE0980 = "XTDE0980" // xsl:number value is NaN or negative
	errCodeXTDE1030 = "XTDE1030" // sort keys have incompatible types
	errCodeXTDE1035 = "XTDE1035" // unknown collation URI
	errCodeXTDE1061 = "XTDE1061" // current-group() called outside for-each-group
	errCodeXTDE1071 = "XTDE1071" // current-grouping-key unavailable
	errCodeXTDE1110 = "XTDE1110" // circular variable reference
	errCodeXTDE1140 = "XTDE1140" // xsl:analyze-string regex error
	errCodeXTDE1145 = "XTDE1145" // xsl:analyze-string q flag not allowed in XSLT 2.0
	errCodeXTDE1150 = "XTDE1150" // xsl:analyze-string regex matches zero-length string
	errCodeXTDE1160 = "XTDE1160" // fragment identifier in document URI not supported
	errCodeXTDE1162 = "XTDE1162" // document() with text node without base URI
	errCodeXTDE1170 = "XTDE1170" // key function lookup error
	errCodeXTDE1270 = "XTDE1270" // key() applied to non-document tree
	errCodeXTDE1360 = "XTDE1360" // current() called when context item is absent
	errCodeXTDE1370 = "XTDE1370" // unparsed-entity-uri with no context node or non-document root
	errCodeXTDE1380 = "XTDE1380" // unparsed-entity-public-id with no context node or non-document root
	errCodeXTDE1390 = "XTDE1390" // system-property with invalid QName or undeclared prefix
	errCodeXTDE1400 = "XTDE1400" // invalid QName in function-available checks
	errCodeXTDE1428 = "XTDE1428" // type-available with invalid EQName or undeclared prefix
	errCodeXTDE1440 = "XTDE1440" // element-available with invalid EQName or undeclared prefix
	errCodeXTDE1450 = "XTDE1450" // no xsl:fallback for forwards-compatible instruction
	errCodeXTDE1460 = "XTDE1460" // xsl:result-document format does not match any declared xsl:output
	errCodeXTDE1480 = "XTDE1480" // xsl:result-document not allowed in temporary output state
	errCodeXTDE1490 = "XTDE1490" // two result documents with same URI
	errCodeXTDE1665 = "XTDE1665" // validation of copy fails
	errCodeXTDE2210 = "XTDE2210" // merge input inconsistent or not sorted
	errCodeXTDE3052 = "XTDE3052" // runtime: abstract component invoked
	errCodeXTDE3086 = "XTDE3086" // required global context item absent
	errCodeXTDE3160 = "XTDE3160" // xsl:evaluate runtime error
	errCodeXTDE3340 = "XTDE3340" // unknown accumulator name
	errCodeXTDE3350 = "XTDE3350" // accumulator-before/after called via dynamic function reference
	errCodeXTDE3362 = "XTDE3362" // streaming accumulator error
	errCodeXTDE3365 = "XTDE3365" // duplicate key in xsl:map construction
	errCodeXTDE3400 = "XTDE3400" // cyclic accumulator dependency
	errCodeXTDE3480 = "XTDE3480" // current-merge-group unavailable outside merge-action
	errCodeXTDE3490 = "XTDE3490" // current-merge-group invalid source name
	errCodeXTDE3510 = "XTDE3510" // current-merge-key unavailable outside merge-action
	errCodeXTDE3530 = "XTDE3530" // xsl:try output cannot be rolled back
	errCodeXTDE3630 = "XTDE3630" // source-document runtime error
	errCodeXTDE3635 = "XTDE3635" // source-document URI resolution error

	// XTMM — Message/Assert
	errCodeXTMM9000 = "XTMM9000" // xsl:message terminate default error code
	errCodeXTMM9001 = "XTMM9001" // xsl:assert default error code

	// XTRE — Runtime Errors (recoverable)
	errCodeXTRE0540 = "XTRE0540" // attribute after child content
	errCodeXTRE1495 = "XTRE1495" // primary output URI conflict (implicit vs explicit)

	// XTSE — Static Errors
	errCodeXTSE0010 = "XTSE0010" // generic static error
	errCodeXTSE0020 = "XTSE0020" // reserved namespace
	errCodeXTSE0080 = "XTSE0080" // duplicate template name
	errCodeXTSE0090 = "XTSE0090" // unknown XSLT element
	errCodeXTSE0110 = "XTSE0110" // required attribute missing
	errCodeXTSE0120 = "XTSE0120" // unsupported version
	errCodeXTSE0125 = "XTSE0125" // duplicate package version with same package-version
	errCodeXTSE0130 = "XTSE0130" // duplicate global variable
	errCodeXTSE0150 = "XTSE0150" // xsl:import-schema after non-import children
	errCodeXTSE0165 = "XTSE0165" // invalid xpath expression
	errCodeXTSE0210 = "XTSE0210" // circular import
	errCodeXTSE0215 = "XTSE0215" // type attribute requires schema-aware processor
	errCodeXTSE0220 = "XTSE0220" // validation attribute requires schema-aware processor
	errCodeXTSE0260 = "XTSE0260" // element required to be empty has content
	errCodeXTSE0265 = "XTSE0265" // conflicting input-type-annotations across modules
	errCodeXTSE0270 = "XTSE0270" // conflicting strip-space/preserve-space
	errCodeXTSE0280 = "XTSE0280" // element not allowed in context
	errCodeXTSE0340 = "XTSE0340" // unknown attribute on XSLT element
	errCodeXTSE0350 = "XTSE0350" // value-of with content and select
	errCodeXTSE0500 = "XTSE0500" // invalid pattern
	errCodeXTSE0530 = "XTSE0530" // priority is not a valid xs:decimal
	errCodeXTSE0545 = "XTSE0545" // conflicting xsl:mode declarations
	errCodeXTSE0550 = "XTSE0550" // invalid mode name on xsl:template
	errCodeXTSE0580 = "XTSE0580" // invalid avt / duplicate parameter name
	errCodeXTSE0590 = "XTSE0590" // as= references undeclared schema type/element/attribute
	errCodeXTSE0620 = "XTSE0620" // cannot write to output
	errCodeXTSE0630 = "XTSE0630" // duplicate global variable or parameter
	errCodeXTSE0670 = "XTSE0670" // duplicate with-param name in xsl:next-iteration
	errCodeXTSE0680 = "XTSE0680" // invalid xsl:number
	errCodeXTSE0690 = "XTSE0690" // required parameter not supplied in call-template
	errCodeXTSE0710 = "XTSE0710" // undeclared attribute-set reference
	errCodeXTSE0720 = "XTSE0720" // cyclic use-attribute-sets
	errCodeXTSE0730 = "XTSE0730" // xsl:attribute-set circular reference
	errCodeXTSE0760 = "XTSE0760" // conflicting xsl:mode property values
	errCodeXTSE0770 = "XTSE0770" // override type mismatch
	errCodeXTSE0800 = "XTSE0800" // reserved namespace used as extension namespace
	errCodeXTSE0805 = "XTSE0805" // conflicting namespace binding on LRE
	errCodeXTSE0808 = "XTSE0808" // invalid combination of xsl:on-empty/xsl:on-non-empty
	errCodeXTSE0809 = "XTSE0809" // #default in exclude-result-prefixes with no default namespace
	errCodeXTSE0810 = "XTSE0810" // conflicting namespace-alias declarations
	errCodeXTSE0840 = "XTSE0840" // xsl:attribute with select and non-empty content
	errCodeXTSE0910 = "XTSE0910" // xsl:namespace select/content exclusivity
	errCodeXTSE0940 = "XTSE0940" // select+content exclusivity on comment/PI
	errCodeXTSE0975 = "XTSE0975" // xsl:number value with select/level/count/from
	errCodeXTSE1015 = "XTSE1015" // xsl:sort with select must be empty
	errCodeXTSE1017 = "XTSE1017" // stable attribute only on first xsl:sort
	errCodeXTSE1040 = "XTSE1040" // xsl:merge-source child position constraint
	errCodeXTSE1060 = "XTSE1060" // for-each-group missing grouping attribute
	errCodeXTSE1070 = "XTSE1070" // for-each-group multiple grouping attributes
	errCodeXTSE1080 = "XTSE1080" // for-each-group invalid attribute combination
	errCodeXTSE1090 = "XTSE1090" // xsl:merge-source must be empty
	errCodeXTSE1130 = "XTSE1130" // xsl:analyze-string missing matching branches
	errCodeXTSE1205 = "XTSE1205" // xsl:key use/content exclusivity
	errCodeXTSE1210 = "XTSE1210" // invalid collation URI on xsl:key
	errCodeXTSE1222 = "XTSE1222" // conflicting xsl:key composite attribute values
	errCodeXTSE1290 = "XTSE1290" // conflicting decimal-format declarations
	errCodeXTSE1295 = "XTSE1295" // zero-digit is not a Unicode digit-zero character
	errCodeXTSE1300 = "XTSE1300" // decimal-format characters conflict (same char for two roles)
	errCodeXTSE1430 = "XTSE1430" // xsl:on-empty/xsl:on-non-empty ordering constraint
	errCodeXTSE1560 = "XTSE1560" // conflicting xsl:output declarations
	errCodeXTSE1570 = "XTSE1570" // invalid output method
	errCodeXTSE1580 = "XTSE1580" // duplicate character-map declaration with same name
	errCodeXTSE1590 = "XTSE1590" // unresolved character map reference
	errCodeXTSE1600 = "XTSE1600" // circular character-map reference
	errCodeXTSE2200 = "XTSE2200" // xsl:merge-source child merge-key count mismatch
	errCodeXTSE3000 = "XTSE3000" // package not found
	errCodeXTSE3005 = "XTSE3005" // circular package dependency
	errCodeXTSE3008 = "XTSE3008" // override not allowed for this component
	errCodeXTSE3010 = "XTSE3010" // xsl:expose component not in this package
	errCodeXTSE3020 = "XTSE3020" // xsl:expose visibility error
	errCodeXTSE3022 = "XTSE3022" // xsl:expose would increase visibility
	errCodeXTSE3025 = "XTSE3025" // xsl:expose conflict
	errCodeXTSE3030 = "XTSE3030" // xsl:accept visibility error
	errCodeXTSE3032 = "XTSE3032" // xsl:accept conflict
	errCodeXTSE3040 = "XTSE3040" // using a hidden component
	errCodeXTSE3050 = "XTSE3050" // accept of non-existent component
	errCodeXTSE3051 = "XTSE3051" // accept makes component private but it is also overridden
	errCodeXTSE3055 = "XTSE3055" // override homonymous with local declaration
	errCodeXTSE3058 = "XTSE3058" // override child no matching component
	errCodeXTSE3060 = "XTSE3060" // conflicting accept rules
	errCodeXTSE3070 = "XTSE3070" // overriding a final component
	errCodeXTSE3080 = "XTSE3080" // overriding a private component
	errCodeXTSE3085 = "XTSE3085" // declared mode is used without xsl:mode declaration
	errCodeXTSE3087 = "XTSE3087" // duplicate xsl:global-context-item declaration
	errCodeXTSE3089 = "XTSE3089" // global-context-item use='absent' with @as
	errCodeXTSE3105 = "XTSE3105" // match pattern element name not in imported schema (typed="strict")
	errCodeXTSE3120 = "XTSE3120" // xsl:break/xsl:next-iteration not allowed in this position
	errCodeXTSE3125 = "XTSE3125" // xsl:break/xsl:on-completion with both select and body
	errCodeXTSE3130 = "XTSE3130" // xsl:next-iteration references undeclared parameter
	errCodeXTSE3140 = "XTSE3140" // xsl:try with select has non-catch/fallback children
	errCodeXTSE3150 = "XTSE3150" // xsl:catch with select has non-empty content
	errCodeXTSE3155 = "XTSE3155" // xsl:function with streamability but no params
	errCodeXTSE3185 = "XTSE3185" // xsl:sequence with select has non-fallback content
	errCodeXTSE3190 = "XTSE3190" // duplicate xsl:merge-source name
	errCodeXTSE3195 = "XTSE3195" // xsl:merge-source has both for-each-source and for-each-item
	errCodeXTSE3200 = "XTSE3200" // xsl:merge-key has both select and sequence constructor
	errCodeXTSE3280 = "XTSE3280" // xsl:map-entry select/body exclusivity
	errCodeXTSE3350 = "XTSE3350" // duplicate accumulator name
	errCodeXTSE3430 = "XTSE3430" // non-streamable expression in streaming context
	errCodeXTSE3441 = "XTSE3441" // streamability error in accumulator
	errCodeXTSE3450 = "XTSE3450" // conflicting static variable values
	errCodeXTSE3470 = "XTSE3470" // merge-key or merge-group disallowed in context
	errCodeXTSE3500 = "XTSE3500" // current-merge-key disallowed in pattern
	errCodeXTSE3520 = "XTSE3520" // iterate param without default when type requires value

	// XTTE — Type Errors
	errCodeXTTE0510 = "XTTE0510" // apply-templates default select without node context
	errCodeXTTE0590 = "XTTE0590" // global context item type mismatch
	errCodeXTTE0945 = "XTTE0945" // xsl:copy with no context item
	errCodeXTTE0990 = "XTTE0990" // xsl:number value is not a positive integer
	errCodeXTTE0950 = "XTTE0950" // namespace-sensitive content copied with copy-namespaces=no + validation=preserve
	errCodeXTTE1000 = "XTTE1000" // xsl:number level/count/from requires a node context
	errCodeXTTE1100 = "XTTE1100" // xsl:merge-source sequence has non-node items
	errCodeXTTE1020 = "XTTE1020" // xsl:merge-key select must return one atomic value
	errCodeXTTE1510 = "XTTE1510" // validation of constructed element fails
	errCodeXTTE1512 = "XTTE1512" // no matching element declaration for strict validation of computed element
	errCodeXTTE1515 = "XTTE1515" // validation of constructed attribute fails
	errCodeXTTE1535 = "XTTE1535" // type annotation not valid for text/comment/PI/attribute nodes
	errCodeXTTE1540 = "XTTE1540" // validation of result document fails (xsl:type on element/attribute)
	errCodeXTTE1550 = "XTTE1550" // document node is not a valid document (multiple roots or text nodes)
	errCodeXTTE1555 = "XTTE1555" // attribute validation fails (no matching global attribute declaration)
	errCodeXTTE2230 = "XTTE2230" // merge keys are not comparable
	errCodeXTTE3100 = "XTTE3100" // typed mode: untyped node in typed mode
	errCodeXTTE3110 = "XTTE3110" // typed="no" mode: typed node in untyped mode
	errCodeXTTE3165 = "XTTE3165" // xsl:evaluate with-params map key is not xs:QName
	errCodeXTTE3170 = "XTTE3170" // xsl:evaluate namespace-context not a single node
	errCodeXTTE3180 = "XTTE3180" // type error in streaming
	errCodeXTTE3210 = "XTTE3210" // xsl:evaluate context-item is not a single item
	errCodeXTTE3360 = "XTTE3360" // accumulator-before/after with non-node context
)

// Sentinel errors for the xslt3 package.
var (
	ErrStaticError   = errors.New("xslt3: static error")
	ErrDynamicError  = errors.New("xslt3: dynamic error")
	ErrCircularRef   = errors.New("xslt3: circular reference")
	ErrNoTemplate    = errors.New("xslt3: no matching template")
	ErrTerminated    = errors.New("xslt3: terminated by xsl:message")
	ErrInvalidOutput = errors.New("xslt3: invalid output specification")

	errNilStylesheet      = errors.New("xslt3: nil stylesheet")
	errNilDocument        = errors.New("xslt3: nil document")
	errZeroInvocation     = errors.New("xslt3: unconfigured invocation")
)

// XSLTError is a structured error with an XSLT error code.
type XSLTError struct {
	Code    string
	Message string
	Cause   error
	// Value holds the sequence associated with the error. For xsl:message
	// terminate="yes", this carries the message body content (nodes/atomics)
	// so that $err:value in xsl:catch can expose it.
	Value any
}

func (e *XSLTError) Error() string {
	if e.Code != "" {
		return e.Code + ": " + e.Message
	}
	return e.Message
}

func (e *XSLTError) Unwrap() error {
	return e.Cause
}

func staticError(code, format string, args ...any) *XSLTError {
	return &XSLTError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Cause:   ErrStaticError,
	}
}

func dynamicError(code, format string, args ...any) *XSLTError {
	return &XSLTError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Cause:   ErrDynamicError,
	}
}

// isXSLTError returns true if err is an XSLTError with the given code.
func isXSLTError(err error, code string) bool {
	if xe, ok := errors.AsType[*XSLTError](err); ok {
		return xe.Code == code
	}
	return false
}

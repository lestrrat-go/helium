package xslt3

import (
	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
)

const (
	// Mode sentinel values per the XSLT 3.0 specification.
	modeDefault = "#default" // the default (unnamed) mode, used as internal key
	modeUnnamed = "#unnamed" // explicit unnamed mode reference
	modeCurrent = "#current" // inherit current mode from caller
	modeAll     = "#all"     // template applies to all modes

	// Output method names per the XSLT 3.0 serialization specification.
	methodXML      = "xml"
	methodHTML     = "html"
	methodXHTML    = "xhtml"
	methodText     = "text"
	methodJSON     = "json"
	methodAdaptive = "adaptive"

	// on-no-match behavior values for xsl:mode.
	onNoMatchDeepCopy      = "deep-copy"
	onNoMatchDeepSkip      = "deep-skip"
	onNoMatchShallowCopy   = "shallow-copy"
	onNoMatchShallowSkip   = "shallow-skip"
	onNoMatchTextOnlyCopy  = "text-only-copy"
	onNoMatchFail          = "fail"

	// Serialization parameter names (xsl:output / xsl:result-document attributes).
	paramMethod               = "method"
	paramVersion              = "version"
	paramEncoding             = "encoding"
	paramOmitXMLDeclaration   = "omit-xml-declaration"
	paramStandalone           = "standalone"
	paramDoctypePublic        = "doctype-public"
	paramDoctypeSystem        = "doctype-system"
	paramCDATASectionElements = "cdata-section-elements"
	paramIndent               = "indent"
	paramMediaType            = "media-type"
	paramByteOrderMark        = "byte-order-mark"
	paramEscapeURIAttributes  = "escape-uri-attributes"
	paramIncludeContentType   = "include-content-type"
	paramNormalizationForm    = "normalization-form"
	paramUndeclarePrefixes    = "undeclare-prefixes"
	paramUseCharacterMaps     = "use-character-maps"
	paramSuppressIndentation  = "suppress-indentation"
	paramHTMLVersion          = "html-version"
	paramItemSeparator        = "item-separator"
	paramJSONNodeOutputMethod = "json-node-output-method"
	paramParameterDocument    = "parameter-document"
	paramBuildTree            = "build-tree"
	paramAllowDuplicateNames  = "allow-duplicate-names"

	// Validation mode values for validation attribute on xsl:copy, xsl:element, etc.
	validationStrict      = "strict"
	validationLax         = "lax"
	validationPreserve    = "preserve"
	validationStrip       = "strip"
	validationUnspecified = "unspecified"

	// Context item use values for xsl:context-item/@use.
	ctxItemRequired = "required"
	ctxItemOptional = "optional"
	ctxItemAbsent   = "absent"

	// on-multiple-match values for xsl:mode.
	onMultipleMatchUseLast = "use-last"
)

// funcKey identifies an xsl:function by its expanded QName and arity.
type funcKey struct {
	Name  xpath3.QualifiedName
	Arity int
}

// Stylesheet is a compiled XSLT stylesheet ready for transformation.
type Stylesheet struct {
	version             string
	templates           []*template
	namedTemplates      map[string]*template
	modeTemplates       map[string][]*template // mode -> templates sorted by import-precedence then priority
	defaultMode         string
	globalVars          []*variable // topologically sorted
	globalParams        []*param
	keys                map[string][]*keyDef
	outputs             map[string]*OutputDef                 // "" = default output
	functions           map[funcKey]*xslFunction // xsl:function defs (keyed by name+arity)
	stripSpace          []nameTest
	preserveSpace       []nameTest
	namespaces          map[string]string                             // prefix -> URI from stylesheet
	excludePrefixes     map[string]struct{}                           // prefixes excluded from output
	excludeURIs         map[string]struct{}                           // namespace URIs excluded from output
	decimalFormats      map[xpath3.QualifiedName]xpath3.DecimalFormat // named decimal formats
	decimalFmtPrec      map[xpath3.QualifiedName]int                  // import precedence of each decimal format
	decimalFmtSet       map[xpath3.QualifiedName]map[string]struct{}  // explicitly set properties per format
	decimalFmtConflicts map[xpath3.QualifiedName]int                  // pending XTSE1290 conflicts (value = precedence)
	modeDefs            map[string]*modeDef                           // mode name -> mode definition
	attributeSets       map[string]*attributeSetDef                   // xsl:attribute-set definitions
	accumulators        map[string]*accumulatorDef                    // accumulator name -> definition
	accumulatorOrder    []string
	sourceDoc           *helium.Document            // the parsed stylesheet document (for document(""))
	moduleDocs          map[string]*helium.Document // module base URI -> parsed stylesheet document
	baseURI             string                      // base URI for resolving relative document references
	schemas             []*xsd.Schema               // imported schemas (xsl:import-schema)
	schemaAware         bool                        // true when xsl:import-schema was encountered
	defaultValidation    string // "strict", "lax", "preserve", "strip" (default-validation attr)
	inputTypeAnnotations string // "preserve", "strip", "unspecified" (input-type-annotations attr)
	defaultCollation    string                      // default-collation URI from stylesheet root
	namespaceAliases    []namespaceAlias            // xsl:namespace-alias declarations
	isPackage           bool                        // true when compiled from xsl:package root
	packageName         string                      // xsl:package/@name
	packageVersion      string                      // xsl:package/@package-version
	declaredModes       bool                        // xsl:package/@declared-modes (default true)
	usedPackages        []*Stylesheet               // packages loaded via xsl:use-package
	// Visibility maps track per-component visibility for package system.
	// Keys are component names (expanded QNames for templates/variables,
	// QualifiedName.String() for functions with arity suffix).
	templateVisibility    map[string]string     // template name -> visibility
	functionVisibility    map[string]string     // "ns:local#arity" -> visibility
	variableVisibility    map[string]string     // variable name -> visibility
	attrSetVisibility     map[string]string     // attribute-set name -> visibility
	globalParamVisibility map[string]string     // global param name -> visibility
	globalContextItem     *globalContextItemDef // xsl:global-context-item declaration
	globalContextModules  map[string]*globalContextItemDef
	characterMaps         map[string]*characterMapDef // name -> character map definition
	packageResolver       PackageResolver             // resolver used at compile time (for fn:transform package-name)
	uriResolver           URIResolver                 // resolver used at compile time (for fn:transform nested compiles)
	compilerImportSchemas []*xsd.Schema               // pre-compiled schemas from compiler (for fn:transform nested compiles)
}

// globalContextItemDef represents a compiled xsl:global-context-item declaration.
type globalContextItemDef struct {
	Use string // "required", "optional", "absent"
	As  string // sequence type (e.g., "document-node(element(codd))")
}

// DefaultOutputDef returns the default output definition for the stylesheet
// as declared at compile time. For the effective output definition that
// includes runtime overrides from xsl:result-document, use
// Invocation.ResolvedOutputDef after calling Do.
func (s *Stylesheet) DefaultOutputDef() *OutputDef {
	if s == nil {
		return nil
	}
	return s.outputs[""]
}

// defaultItemSeparator returns the item-separator from the default output
// definition, or nil if not set.
func (s *Stylesheet) defaultItemSeparator() *string {
	if s == nil {
		return nil
	}
	if outDef, ok := s.outputs[""]; ok {
		return outDef.ItemSeparator
	}
	return nil
}

// namespaceAlias maps a stylesheet namespace URI to a result namespace URI.
type namespaceAlias struct {
	StylesheetURI string // namespace URI used in stylesheet
	ResultURI     string // namespace URI to use in output
	ResultPrefix  string // preferred prefix for the result namespace
	ImportPrec    int    // import precedence for conflict resolution
}

// modeDef is a compiled xsl:mode declaration.
type modeDef struct {
	Name            string
	OnNoMatch       string // "shallow-copy", "deep-copy", "shallow-skip", "deep-skip", "text-only-copy", "fail"
	Typed           string // "strict", "lax", "unspecified", "yes", "no"
	Streamable      bool
	Visibility      string // "public", "private", "final"
	OnMultipleMatch string // "use-last", "fail"
	UseAccumulators *string // nil = attribute absent; non-nil = attribute present (space-separated names, "#all", or "")
	ImportPrec          int
	conflictStreamable  bool // deferred: conflicting streamable at same prec
	conflictOnNoMatch   bool
	conflictVisibility  bool
	conflictOnMultiple  bool
	conflictAccumulator bool
}

// attributeSetDef is a compiled xsl:attribute-set.
type attributeSetDef struct {
	Name        string
	UseAttrSets []string      // names of other attribute sets to include
	Attrs       []instruction // xsl:attribute instructions
	Visibility  string        // "public", "private", "final", "abstract"
	Streamable  bool          // streamable="yes" on the attribute-set
}

// xslFunction is a compiled xsl:function.
type xslFunction struct {
	Name          xpath3.QualifiedName
	Params        []*param
	Body          []instruction
	As            string      // return type constraint (e.g., "xs:string?")
	Cache         bool        // cache="yes" encourages result memoization by argument values
	Streamability string      // "absorbing", "inspection", etc.; "" = unspecified
	Visibility    string      // "public", "private", "final", "abstract"
	NewEachTime   string      // "yes", "no", "maybe"; "" = unspecified (defaults to "maybe")
	OwnerPackage  *Stylesheet   // package that defined this function (nil = main stylesheet)
	ImportPrec    int           // import precedence for XTSE0770 conflict detection
	OriginalFunc  *xslFunction // original function being overridden (for xsl:original calls)
	IsOverride    bool         // true if this function was defined in xsl:override
}

// template is a compiled xsl:template.
type template struct {
	Match            *pattern
	Name             string
	Mode             string // "" = default, "#all" = all modes
	Priority         float64
	ImportPrec       int
	MinImportPrec    int // lowest import precedence among this module's imports (for apply-imports boundary)
	Params           []*param
	Body             []instruction
	As               string      // return type constraint (e.g., "element()")
	ContextItemAs    string      // xsl:context-item/@as type constraint
	ContextItemUse   string      // xsl:context-item/@use: "required" (default), "optional", "absent"
	XPathDefaultNS   string      // xpath-default-namespace (inherited or explicit)
	DefaultCollation string      // default-collation URI (inherited or explicit)
	BaseURI          string      // base URI of the stylesheet module that defined this template
	Visibility       string      // "public", "private", "final", "abstract"
	Version          string      // effective version (from template or stylesheet)
	OwnerPackage     *Stylesheet // package that defined this template (nil = main stylesheet)
	OriginalTemplate *template   // original template being overridden (for xsl:original calls)
}

// variable is a compiled xsl:variable.
type variable struct {
	Name             string
	Select           *xpath3.Expression
	Body             []instruction  // used when select is absent
	As               string         // type declaration (e.g., "element()*")
	Visibility       string         // "public", "private", "final", "abstract"
	OwnerPackage     *Stylesheet    // package that defined this variable (nil = main stylesheet)
	OriginalVar      *variable      // original variable being overridden (for $xsl:original)
	ImportPrec       int            // import precedence for XTSE0630 duplicate detection
	StaticValue      xpath3.Sequence // pre-computed value for static="yes" variables
	XPathDefaultNS   string         // xpath-default-namespace in scope at definition site
}

// param is a compiled xsl:param.
type param struct {
	Name       string
	Select     *xpath3.Expression
	Body       []instruction // used when select is absent
	As         string        // type declaration (e.g., "xs:integer")
	Required   bool
	Tunnel     bool
	Visibility string // "public", "private", "final", "abstract" (for global params)
	ImportPrec int    // import precedence for XTSE0630 duplicate detection
}

// keyDef is a compiled xsl:key.
type keyDef struct {
	Name      string
	Match     *pattern
	Use       *xpath3.Expression
	Body      []instruction // content constructor (when use attribute is absent)
	Composite bool          // composite="yes" on xsl:key
	Collation string        // collation URI (explicit or default; empty = codepoint)
}

// OutputDef is a compiled xsl:output.
type OutputDef struct {
	Name              string
	Method            string // "xml", "html", "text", "xhtml"
	MethodExplicit    bool   // true when method was explicitly specified (not defaulted)
	Encoding          string
	Indent            bool
	OmitDeclaration         bool
	OmitDeclarationExplicit bool // true when omit-xml-declaration was explicitly set
	Standalone        string // "yes", "no", "omit"
	CDATASections     []string
	DoctypePublic     string
	DoctypeSystem     string
	MediaType          string
	Version            string
	UndeclarePrefixes  bool
	IncludeContentType *bool   // include-content-type: nil=default(yes), true/false
	ItemSeparator      *string // item-separator serialization parameter; nil = not set
	ItemSeparatorAbsent bool   // true when item-separator="#absent" was explicitly set
	HTMLVersion        string            // html-version: "5", "4.0", etc.
	NormalizationForm  string            // "NFC", "NFD", "NFKC", "NFKD", "fully-normalized", "none"
	ByteOrderMark      bool             // byte-order-mark: emit BOM at start of output
	EscapeURIAttributes *bool           // escape-uri-attributes: nil=default(true), true/false
	UseCharacterMaps    []string        // names of character maps to use
	ResolvedCharMap     map[rune]string  // resolved character map (populated at runtime)
	AllowDuplicateNames  bool             // allow-duplicate-names for JSON output
	JSONNodeOutputMethod string            // json-node-output-method: "xml", "html", "xhtml", "text"
	SuppressIndentation []string         // suppress-indentation element names
	ParameterDocument   string           // parameter-document URI
	BuildTree           *bool            // build-tree: nil=default(true), true/false
	ImportPrec         int              // import precedence for XTSE1560 conflict detection
	// Raw attribute values for XTSE1560 conflict detection
	MethodRaw      string
	IndentRaw      string
	EncodingRaw    string
	VersionRaw     string
	StandaloneRaw  string
}

// GetUseCharacterMaps returns the use-character-maps list, nil-safe.
func (o *OutputDef) GetUseCharacterMaps() []string {
	if o == nil {
		return nil
	}
	return o.UseCharacterMaps
}

// characterMapDef is a compiled xsl:character-map.
type characterMapDef struct {
	Name             string            // expanded QName
	Mappings         map[rune]string   // character -> replacement string
	UseCharacterMaps []string          // other character maps to include
}

// accumulatorDef is a compiled xsl:accumulator.
type accumulatorDef struct {
	Name              string
	As                string // type declaration
	Initial           *xpath3.Expression
	InitialBody       []instruction
	Rules             []*accumulatorRule
	Streamable        bool
	ImportPrec        int  // import precedence for conflict resolution
	FromPackage       bool // true if imported via xsl:use-package
	conflictDuplicate bool // deferred XTSE3350: same-precedence duplicate
	CyclicDeps        bool // XTDE3400: cyclic dependency detected
}

// accumulatorRule is a compiled xsl:accumulator-rule.
type accumulatorRule struct {
	Match  *pattern
	Phase  string // "start" or "end"
	Select *xpath3.Expression
	Body   []instruction
	New    bool // new="yes" starts a fresh value
}

// nameTest is used for xsl:strip-space and xsl:preserve-space element names.
type nameTest struct {
	Prefix string
	Local  string // "*" = wildcard
}

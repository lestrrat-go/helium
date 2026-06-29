package xsd

import (
	"slices"

	"github.com/lestrrat-go/helium/internal/xsdregex"
	"github.com/lestrrat-go/helium/xpath1"
	"github.com/lestrrat-go/helium/xpath3"
)

// QName represents a namespace-qualified name.
type QName struct {
	Local string
	NS    string
}

// BlockFlags is a bitmask for the block attribute on element declarations.
type BlockFlags uint8

// BlockFlags values.
const (
	BlockExtension BlockFlags = 1 << iota
	BlockRestriction
	BlockSubstitution
)

// FinalFlags is a bitmask for the final attribute on element/type declarations.
type FinalFlags uint8

// FinalFlags values.
const (
	FinalExtension FinalFlags = 1 << iota
	FinalRestriction
	FinalList  // simpleType only
	FinalUnion // simpleType only
)

// Schema represents a compiled XML Schema.
// (libxml2: xmlSchema)
type Schema struct {
	// version is the effective XSD specification version resolved at compile
	// time (explicit Compiler.Version() or a vc:minVersion hint). The Validator
	// reads it to apply the same version-specific semantics as compilation.
	version           Version
	targetNamespace   string
	elemFormQualified bool // elementFormDefault="qualified"
	attrFormQualified bool // attributeFormDefault="qualified"
	blockDefault      BlockFlags
	finalDefault      FinalFlags
	elements          map[QName]*ElementDecl
	types             map[QName]*TypeDef
	groups            map[QName]*ModelGroup
	attrGroups        map[QName][]*AttrUse
	globalAttrs       map[QName]*AttrUse
	substGroups       map[QName][]*ElementDecl // head QName → member element declarations
}

// LookupElement returns the global element declaration for the given name.
func (s *Schema) LookupElement(local, ns string) (*ElementDecl, bool) {
	e, ok := s.elements[QName{Local: local, NS: ns}]
	return e, ok
}

// LookupType returns the type definition for the given name.
func (s *Schema) LookupType(local, ns string) (*TypeDef, bool) {
	t, ok := s.types[QName{Local: local, NS: ns}]
	return t, ok
}

// SubstGroupMembers returns the element declarations in the substitution group
// of the given head element.
func (s *Schema) SubstGroupMembers(head QName) []*ElementDecl {
	return s.substGroups[head]
}

// LookupAttribute returns the global attribute declaration for the given name.
func (s *Schema) LookupAttribute(local, ns string) (*AttrUse, bool) {
	a, ok := s.globalAttrs[QName{Local: local, NS: ns}]
	return a, ok
}

// NamedTypes returns the schema's named type definitions.
func (s *Schema) NamedTypes() []QName {
	if len(s.types) == 0 {
		return nil
	}
	names := make([]QName, 0, len(s.types))
	for qn := range s.types {
		names = append(names, qn)
	}
	slices.SortFunc(names, func(a, b QName) int {
		if a.NS != b.NS {
			if a.NS < b.NS {
				return -1
			}
			return 1
		}
		if a.Local < b.Local {
			return -1
		}
		if a.Local > b.Local {
			return 1
		}
		return 0
	})
	return names
}

// TargetNamespace returns the schema's target namespace.
func (s *Schema) TargetNamespace() string {
	return s.targetNamespace
}

// ContentTypeKind describes the content type of a complex type.
type ContentTypeKind int

// ContentTypeKind values.
const (
	ContentTypeEmpty       ContentTypeKind = iota // element has no content
	ContentTypeSimple                             // text-only content
	ContentTypeElementOnly                        // child elements only (no mixed text)
	ContentTypeMixed                              // elements + text interleaved
)

// ModelGroupKind describes the compositor of a model group.
type ModelGroupKind int

// ModelGroupKind values.
const (
	CompositorSequence ModelGroupKind = iota // xs:sequence
	CompositorChoice                         // xs:choice
	CompositorAll                            // xs:all
)

// Unbounded is the sentinel for maxOccurs="unbounded".
const Unbounded = -1

// ElementDecl is a schema element declaration.
type ElementDecl struct {
	Name              QName
	Type              *TypeDef
	MinOccurs         int
	MaxOccurs         int // -1 = unbounded
	Abstract          bool
	Nillable          bool  // true if the element may carry xsi:nil="true"
	SubstitutionGroup QName // QName of the (first) substitution group head (zero value if none)
	// SubstitutionGroups holds ALL heads when XSD 1.1 multiple-head substitution
	// is used (substitutionGroup="a b c"). It is nil for the common single-head
	// case; SubstitutionGroup always holds the first head for back-compat.
	SubstitutionGroups []QName
	IsRef              bool // true if this was created from a ref="..." attribute
	IDCs               []*IDConstraint
	Default            *string // nil = not set
	Fixed              *string // nil = not set
	// FixedNS holds the in-scope namespace bindings (prefix → URI) at the point
	// the Fixed value was declared in the schema document. It is used to resolve
	// a QName/NOTATION fixed value's prefix when comparing in value space.
	FixedNS map[string]string
	// DefaultNS mirrors FixedNS for the Default value: a QName/NOTATION default
	// substituted into an empty element resolves its prefix against the
	// DECLARATION's namespace context, not the instance's.
	DefaultNS map[string]string
	Block     BlockFlags
	BlockSet  bool // true if block was explicitly set (even to empty)
	Final     FinalFlags
	FinalSet  bool // true if final was explicitly set (even to empty)
	// Alternatives is the XSD 1.1 conditional-type-assignment {type table}: the
	// ordered xs:alternative children. At validation (when no xsi:type is present)
	// the governing type is the one selected by the first alternative whose @test
	// is true, or a testless default; empty when the declaration has none.
	Alternatives []*TypeAlternative
}

// TypeAlternative is one XSD 1.1 <xs:alternative> in an element declaration's
// conditional type assignment. Test is the XPath 3.1 condition (empty for the
// final, testless default alternative); Type is the governing type selected when
// the test holds. It mirrors Assertion's capture pattern.
type TypeAlternative struct {
	Test       string
	Namespaces map[string]string  // prefix → URI from the schema document
	Line       int                // source line of the xs:alternative element
	Source     string             // source filename of the declaring schema document
	BaseURI    string             // schema document URI, exposed as the XPath static base URI
	TypeName   QName              // the @type reference (resolved into Type during resolveRefs)
	Type       *TypeDef           // resolved governing type
	compiled   *xpath3.Expression // pre-compiled @test; nil for a testless default (or compile failure)
}

// IDCKind describes the kind of identity constraint.
type IDCKind int

// IDCKind values.
const (
	IDCUnique IDCKind = iota // xs:unique
	IDCKey                   // xs:key
	IDCKeyRef                // xs:keyref
)

// IDConstraint represents an xs:unique, xs:key, or xs:keyref identity constraint.
type IDConstraint struct {
	Name       string
	QName      QName // namespace-qualified constraint name ({targetNamespace}Name)
	Kind       IDCKind
	Selector   string            // XPath selector expression
	Fields     []string          // XPath field expressions
	Refer      string            // for keyref: the lexical refer QName as written
	ReferQName QName             // for keyref: the resolved {ns}local of the referenced key/unique
	Namespaces map[string]string // prefix → URI from the schema document (for XPath evaluation)
	Line       int               // source line of the constraint element (for error reporting)
	Source     string            // source filename of the declaring schema document (for error reporting); paired with Line so a deferred @refer error on an IMPORTED constraint cites the imported file, not the importing compiler's filename

	SelectorExpr *xpath1.Expression   // pre-compiled selector XPath
	FieldExprs   []*xpath1.Expression // pre-compiled field XPaths (parallel to Fields)

	// SelectorDefaultNS / FieldDefaultNS hold the resolved default element
	// namespace URI (XSD 1.1 @xpathDefaultNamespace) for the selector and each
	// field XPath. Empty means no default (XPath 1.0 unprefixed = no-namespace).
	// FieldDefaultNS is parallel to Fields.
	SelectorDefaultNS string
	FieldDefaultNS    []string

	// IsConstraintRef marks an XSD 1.1 identity-constraint that uses @ref to point
	// at another constraint instead of declaring its own name/selector/field. At
	// compile time the referenced constraint's selector/fields (and, for keyref,
	// its refer) are copied in, so validation treats it like any other constraint.
	IsConstraintRef    bool
	ConstraintRef      string // lexical @ref QName as written
	ConstraintRefQName QName  // resolved {ns}local of the referenced constraint

	referUnbound         bool // for keyref: @refer used a prefix not bound in scope (already reported)
	constraintRefUnbound bool // for @ref: the ref prefix was not bound in scope (already reported)
}

// Assertion is an XSD 1.1 xs:assert constraint on a complex type: an XPath 3.1
// expression (the @test) evaluated against the element being validated. The
// element is valid against the assertion only if the test's effective boolean
// value is true. It mirrors IDConstraint's capture pattern: the lexical test,
// the in-scope namespace bindings from the schema document, source line/file for
// diagnostics, and the pre-compiled expression.
type Assertion struct {
	Test       string
	Namespaces map[string]string  // prefix → URI from the schema document
	Line       int                // source line of the xs:assert element
	Source     string             // source filename of the declaring schema document
	compiled   *xpath3.Expression // pre-compiled @test; nil if the expression failed to compile
}

// DerivationKind describes how a type is derived from its base.
type DerivationKind int

// DerivationKind values.
const (
	DerivationNone        DerivationKind = iota // no derivation
	DerivationExtension                         // derived by extension
	DerivationRestriction                       // derived by restriction
)

// TypeVariety describes the variety of a simple type definition.
type TypeVariety int

// TypeVariety values.
const (
	TypeVarietyAtomic TypeVariety = iota // atomic simple type
	TypeVarietyList                      // list of atomic values
	TypeVarietyUnion                     // union of simple types
)

// TypeDef is a schema type definition.
type TypeDef struct {
	Name QName
	// IsComplex distinguishes a complex type DEFINITION from a simple one. It is a
	// reliable discriminator independent of ContentType, because a complex type with
	// <xs:simpleContent> also carries ContentType == ContentTypeSimple. Set true by
	// parseComplexType (and for the built-in xs:anyType); a simple type definition
	// (parseSimpleType, simple built-ins, recovery placeholders) leaves it false.
	IsComplex    bool
	ContentType  ContentTypeKind
	ContentModel *ModelGroup
	BaseType     *TypeDef
	Attributes   []*AttrUse
	AnyAttribute *Wildcard
	Derivation   DerivationKind
	Facets       *FacetSet
	Variety      TypeVariety
	ItemType     *TypeDef   // for list types: the item type definition
	MemberTypes  []*TypeDef // for union types: the member type definitions
	Abstract     bool
	Final        FinalFlags
	FinalSet     bool         // true if final was explicitly set (even to empty)
	Assertions   []*Assertion // XSD 1.1 xs:assert constraints declared directly on this type
	OpenContent  *OpenContent // XSD 1.1 xs:openContent (nil = none)
	// ContentSimpleType is the effective simple type that constrains the text
	// content of a complexType with simpleContent derived by RESTRICTION (XSD 1.1):
	// either the nested <xs:simpleType> or a synthesized restriction of the base
	// content type carrying the restriction's direct facets. nil means the content
	// is validated against this type's own base chain (extensions, and restrictions
	// with no content-narrowing facets). Used by validateSimpleContent so a
	// restricted simpleContent value is actually checked against its narrowed type.
	ContentSimpleType *TypeDef
	// IsSimpleContent marks a COMPLEX type whose content is simple content (an
	// <xs:simpleContent> extension/restriction), distinguishing it from a plain
	// <xs:simpleType> (both carry ContentType == ContentTypeSimple). The effective
	// content-type walk (effectiveContentSimpleType) recurses through simpleContent
	// complex types and stops at the underlying simpleType/builtin, so a base
	// type's facets/assertions are not skipped and a narrowed content type is
	// inherited through derived simpleContent types.
	IsSimpleContent bool
}

// OpenContentMode is the XSD 1.1 xs:openContent mode.
type OpenContentMode int

// OpenContentMode values.
const (
	OpenContentInterleave OpenContentMode = iota // open wildcard elements may be interleaved with the declared content (default)
	OpenContentSuffix                            // open wildcard elements may appear only after the declared content
)

// OpenContent is an XSD 1.1 xs:openContent: an element wildcard that admits extra
// child elements beyond the declared content model, either interleaved among it
// or as a suffix. mode="none" is represented by a nil *OpenContent on the type.
type OpenContent struct {
	Mode     OpenContentMode
	Wildcard *Wildcard // the xs:any wildcard governing the open content
}

// FacetSet holds facet constraints for a simple type restriction.
type FacetSet struct {
	Enumeration       []string
	EnumerationNS     []map[string]string
	MinInclusive      *string
	MaxInclusive      *string
	MinExclusive      *string
	MaxExclusive      *string
	MinInclusiveFixed bool
	MaxInclusiveFixed bool
	MinExclusiveFixed bool
	MaxExclusiveFixed bool
	// MinInclusiveNS/MaxInclusiveNS/MinExclusiveNS/MaxExclusiveNS hold the
	// in-scope namespace bindings (prefix → URI) captured at the point each
	// individual range-facet bound was declared in the schema document. Each
	// facet element gets its own map because sibling facets in the same
	// <xs:restriction> may declare different prefixes (e.g. minInclusive binds
	// p: while maxInclusive binds q:). They mirror EnumerationNS for the range
	// facets: a namespace-sensitive base type (e.g. xs:QName reached through a
	// union) needs this context to resolve a prefixed bound value like
	// <xs:minInclusive value="p:a"/>. A nil map means no extra bindings were in
	// scope at that facet (the facet was absent or carried no namespace context).
	MinInclusiveNS        map[string]string
	MaxInclusiveNS        map[string]string
	MinExclusiveNS        map[string]string
	MaxExclusiveNS        map[string]string
	TotalDigits           *int
	FractionDigits        *int
	Length                *int
	MinLength             *int
	MaxLength             *int
	ExplicitTimezone      *string
	ExplicitTimezoneFixed bool
	// Patterns holds the <xs:pattern> facets from a single restriction step.
	// Per XSD, patterns in the same step are ORed (a value is valid if it
	// matches any of them); patterns from different derivation steps are ANDed,
	// which is handled by validating each step's FacetSet along the type chain.
	Patterns []string
	// compiledPatterns holds the regexes for Patterns, compiled once at schema
	// compile time and index-aligned with Patterns. A nil entry means the
	// pattern failed to compile and is skipped during validation.
	compiledPatterns []*xsdregex.Regexp
	WhiteSpace       *string
	// Assertions holds the XSD 1.1 <xs:assertion> facets from a single
	// simpleType restriction step. Each is evaluated against the value being
	// validated with $value bound to its typed atomic value (a sequence for a
	// list type). Assertions from different derivation steps are all enforced
	// (ANDed), handled by validateValue walking the base-type chain.
	Assertions []*Assertion
}

// AttrUse represents an attribute use in a complex type definition.
// (libxml2: xmlSchemaAttributeUsePtr)
type AttrUse struct {
	Name       QName
	TypeName   QName
	Type       *TypeDef // anonymous inline <xs:simpleType>, if any
	Required   bool
	Prohibited bool
	// Inheritable is the XSD 1.1 {inheritable} property: when true, an instance of
	// this attribute is contributed to the inherited-attribute set of every
	// descendant element (consulted by conditional type assignment / assertions).
	// InheritableSet records whether inheritable was given explicitly on the use,
	// so a ref use's explicit value wins over the referenced declaration's.
	Inheritable    bool
	InheritableSet bool
	Default        *string // nil = not set
	Fixed          *string // nil = not set
	// FixedNS holds the in-scope namespace bindings (prefix → URI) at the point
	// the Fixed value was declared in the schema document, used to resolve a
	// QName/NOTATION fixed value's prefix when comparing in value space.
	FixedNS map[string]string
	// DefaultNS mirrors FixedNS for the Default value: a QName/NOTATION default
	// materialized onto an absent attribute resolves its prefix against the
	// DECLARATION's namespace context, not the instance's.
	DefaultNS map[string]string
}

// ModelGroup is a content model compositor (sequence, choice, all).
type ModelGroup struct {
	Compositor ModelGroupKind
	Particles  []*Particle
	MinOccurs  int
	MaxOccurs  int
}

// ParticleTerm is an interface satisfied by *ElementDecl, *ModelGroup, and *Wildcard.
type ParticleTerm interface {
	particleTerm()
}

func (*ElementDecl) particleTerm() {}
func (*ModelGroup) particleTerm()  {}
func (*Wildcard) particleTerm()    {}

// Particle is a particle in a content model.
type Particle struct {
	MinOccurs int
	MaxOccurs int // -1 = unbounded
	Term      ParticleTerm
}

// ProcessContentsKind describes how matching elements are validated.
type ProcessContentsKind int

// ProcessContentsKind values.
const (
	ProcessStrict ProcessContentsKind = iota // validate against schema
	ProcessLax                               // validate if declaration found, skip otherwise
	ProcessSkip                              // skip validation entirely
)

// Wildcard namespace constraint tokens used in xs:any/@namespace.
const (
	WildcardNSAny             = "##any"
	WildcardNSOther           = "##other"
	WildcardNSLocal           = "##local"
	WildcardNSTargetNamespace = "##targetNamespace"
	WildcardNSNotAbsent       = "##not-absent"
)

// XSD 1.1 wildcard notQName keyword tokens (xs:any/@notQName, xs:anyAttribute/@notQName).
const (
	WildcardQNameDefined        = "##defined"
	WildcardQNameDefinedSibling = "##definedSibling"
)

// Wildcard represents an xs:any or xs:anyAttribute wildcard.
type Wildcard struct {
	// Namespace constraint: "##any", "##other", "##local",
	// "##targetNamespace", or a space-separated list of URIs.
	Namespace       string
	ProcessContents ProcessContentsKind
	TargetNS        string // schema's target namespace, for resolving ##other/##targetNamespace

	// XSD 1.1 additions (xsd.Version11 only; nil/false in 1.0).
	//
	// NotNamespace is the resolved set of namespace URIs the wildcard EXCLUDES
	// (from @notNamespace). "" represents the absent namespace (##local). When
	// non-nil the wildcard's positive namespace variety is "not these" — it
	// matches any namespace not in the list. @namespace and @notNamespace are
	// mutually exclusive, so when this is set Namespace defaults to ##any.
	NotNamespace []string
	// NotQName is the resolved set of element/attribute QNames the wildcard
	// EXCLUDES (the QName members of @notQName).
	NotQName []QName
	// NotQNameDefined is true when @notQName contains ##defined: the wildcard
	// excludes any name with a global element (xs:any) or attribute
	// (xs:anyAttribute) declaration.
	NotQNameDefined bool
	// NotQNameDefinedSibling is true when @notQName contains ##definedSibling
	// (xs:any only): the wildcard excludes the names of element declarations
	// that are siblings in the same content model. SiblingNames holds those
	// names, resolved after the content model is built.
	NotQNameDefinedSibling bool
	SiblingNames           []QName
}

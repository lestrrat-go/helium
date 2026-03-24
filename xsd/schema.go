package xsd

import (
	"slices"

	"github.com/lestrrat-go/helium/xpath1"
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
	SubstitutionGroup QName // QName of the substitution group head (zero value if none)
	IsRef             bool  // true if this was created from a ref="..." attribute
	IDCs              []*IDConstraint
	Default           *string // nil = not set
	Fixed             *string // nil = not set
	Block             BlockFlags
	BlockSet          bool // true if block was explicitly set (even to empty)
	Final             FinalFlags
	FinalSet          bool // true if final was explicitly set (even to empty)
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
	Kind       IDCKind
	Selector   string            // XPath selector expression
	Fields     []string          // XPath field expressions
	Refer      string            // for keyref: the name of the referenced key/unique
	Namespaces map[string]string // prefix → URI from the schema document (for XPath evaluation)

	SelectorExpr *xpath1.Expression   // pre-compiled selector XPath
	FieldExprs   []*xpath1.Expression // pre-compiled field XPaths (parallel to Fields)
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
	Name         QName
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
	FinalSet     bool // true if final was explicitly set (even to empty)
}

// FacetSet holds facet constraints for a simple type restriction.
type FacetSet struct {
	Enumeration    []string
	EnumerationNS  []map[string]string
	MinInclusive   *string
	MaxInclusive   *string
	MinExclusive   *string
	MaxExclusive   *string
	TotalDigits    *int
	FractionDigits *int
	Length         *int
	MinLength      *int
	MaxLength      *int
	Pattern        *string
	WhiteSpace     *string
}

// AttrUse represents an attribute use in a complex type definition.
// (libxml2: xmlSchemaAttributeUsePtr)
type AttrUse struct {
	Name       QName
	TypeName   QName
	Required   bool
	Prohibited bool
	Default    *string // nil = not set
	Fixed      *string // nil = not set
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

// Wildcard represents an xs:any or xs:anyAttribute wildcard.
type Wildcard struct {
	// Namespace constraint: "##any", "##other", "##local",
	// "##targetNamespace", or a space-separated list of URIs.
	Namespace       string
	ProcessContents ProcessContentsKind
	TargetNS        string // schema's target namespace, for resolving ##other/##targetNamespace
}

package xsd

// QName represents a namespace-qualified name.
type QName struct {
	Local string
	NS    string
}

// Schema represents a compiled XML Schema.
type Schema struct {
	targetNamespace   string
	elemFormQualified bool // elementFormDefault="qualified"
	attrFormQualified bool // attributeFormDefault="qualified"
	elements          map[QName]*ElementDecl
	types             map[QName]*TypeDef
	groups            map[QName]*ModelGroup
	attrGroups        map[QName][]*AttrUse
	globalAttrs       map[QName]*AttrUse
	substGroups       map[QName][]*ElementDecl // head QName → member element declarations
	compileErrors     string                   // accumulated compilation error messages
	compileWarnings   string                   // accumulated compilation warnings
}

// CompileErrors returns any schema compilation error messages
// in libxml2-compatible format. Empty string means no errors.
func (s *Schema) CompileErrors() string {
	return s.compileErrors
}

// CompileWarnings returns any schema compilation warning messages.
func (s *Schema) CompileWarnings() string {
	return s.compileWarnings
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

// TargetNamespace returns the schema's target namespace.
func (s *Schema) TargetNamespace() string {
	return s.targetNamespace
}

// ContentTypeKind describes the content type of a complex type.
type ContentTypeKind int

const (
	ContentTypeEmpty       ContentTypeKind = iota
	ContentTypeSimple                      // text-only content
	ContentTypeElementOnly                 // child elements only (no mixed text)
	ContentTypeMixed                       // elements + text interleaved
)

// ModelGroupKind describes the compositor of a model group.
type ModelGroupKind int

const (
	CompositorSequence ModelGroupKind = iota
	CompositorChoice
	CompositorAll
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
	SubstitutionGroup QName // QName of the substitution group head (zero value if none)
	IsRef             bool  // true if this was created from a ref="..." attribute
	IDCs              []*IDConstraint
}

// IDCKind describes the kind of identity constraint.
type IDCKind int

const (
	IDCUnique IDCKind = iota
	IDCKey
	IDCKeyRef
)

// IDConstraint represents an xs:unique, xs:key, or xs:keyref identity constraint.
type IDConstraint struct {
	Name       string
	Kind       IDCKind
	Selector   string            // XPath selector expression
	Fields     []string          // XPath field expressions
	Refer      string            // for keyref: the name of the referenced key/unique
	Namespaces map[string]string // prefix → URI from the schema document (for XPath evaluation)
}

// DerivationKind describes how a type is derived from its base.
type DerivationKind int

const (
	DerivationNone        DerivationKind = iota
	DerivationExtension
	DerivationRestriction
)

// TypeVariety describes the variety of a simple type definition.
type TypeVariety int

const (
	TypeVarietyAtomic TypeVariety = iota
	TypeVarietyList
	TypeVarietyUnion
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
}

// FacetSet holds facet constraints for a simple type restriction.
type FacetSet struct {
	Enumeration  []string
	MinInclusive *string
	MaxInclusive *string
	TotalDigits  *int
	Length       *int
	MinLength    *int
	MaxLength    *int
	Pattern      *string
}

// AttrUse is a stub for attribute use declarations (future phases).
type AttrUse struct {
	Name       QName
	TypeName   QName
	Required   bool
	Prohibited bool
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

const (
	ProcessStrict ProcessContentsKind = iota
	ProcessLax
	ProcessSkip
)

// Wildcard represents an xs:any or xs:anyAttribute wildcard.
type Wildcard struct {
	// Namespace constraint: "##any", "##other", "##local",
	// "##targetNamespace", or a space-separated list of URIs.
	Namespace       string
	ProcessContents ProcessContentsKind
	TargetNS        string // schema's target namespace, for resolving ##other/##targetNamespace
}

package xmlschema

// QName represents a namespace-qualified name.
type QName struct {
	Local string
	NS    string
}

// Schema represents a compiled XML Schema.
type Schema struct {
	targetNamespace string
	elements        map[QName]*ElementDecl
	types           map[QName]*TypeDef
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
	Name      QName
	Type      *TypeDef
	MinOccurs int
	MaxOccurs int // -1 = unbounded
}

// TypeDef is a schema type definition.
type TypeDef struct {
	Name         QName
	ContentType  ContentTypeKind
	ContentModel *ModelGroup
	BaseType     *TypeDef
	Attributes   []*AttrUse
}

// AttrUse is a stub for attribute use declarations (future phases).
type AttrUse struct {
	Name     QName
	TypeName QName
	Required bool
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

// Wildcard is a stub for xs:any (future phases).
type Wildcard struct {
	Namespace string
}

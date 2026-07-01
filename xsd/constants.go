package xsd

// attrValTrue and attrValQualified are common XML attribute value strings.
const (
	attrValTrue      = "true"
	attrValQualified = "qualified"
)

// typeAnyType is the local name of the built-in xs:anyType (the ur-type), used
// as the default {type definition} for an element declaration with no resolvable
// type.
const typeAnyType = "anyType"

// Builtin simple-type local names that lack a constant in internal/lexicon,
// used by the builtin restriction-derivation hierarchy in link_refs.go.
const (
	typeAnySimpleType = "anySimpleType"
	typeLanguage      = "language"
	typeName          = "Name"
	typeNCName        = "NCName"
	typeNMToken       = "NMTOKEN"
	typeID            = "ID"
	typeEntity        = "ENTITY"
	typeHexBinary     = "hexBinary"
	typeBase64Binary  = "base64Binary"
	typeIDRefs        = "IDREFS"
	typeEntities      = "ENTITIES"
	typeNMTokens      = "NMTOKENS"
)

const (
	elemAll          = "all"
	elemAlternative  = "alternative" // XSD 1.1: xs:alternative (conditional type assignment)
	elemAnnotation   = "annotation"
	elemAny          = "any"
	elemAnyAttribute = "anyAttribute"
	elemAssert       = "assert"      // XSD 1.1: xs:assert on complex types
	elemAssertion    = "assertion"   // XSD 1.1: xs:assertion facet on simple types
	elemOpenContent  = "openContent" // XSD 1.1: xs:openContent on complex types

	elemDefaultOpenContent = "defaultOpenContent" // XSD 1.1: schema-level default open content
	elemAppinfo            = "appinfo"
	elemAttribute          = "attribute"
	elemAttributeGroup     = "attributeGroup"
	elemChoice             = "choice"
	elemComplexContent     = "complexContent"
	elemComplexType        = "complexType"
	elemDocumentation      = "documentation"
	elemElement            = "element"
	elemExtension          = "extension"
	elemField              = "field"
	elemGroup              = "group"
	elemImport             = "import"
	elemInclude            = "include"
	elemKey                = "key"
	elemKeyRef             = "keyref"
	elemList               = "list"
	elemNotation           = "notation"
	elemOverride           = "override" // XSD 1.1: xs:override (override schema document)
	elemRedefine           = "redefine"
	elemRestriction        = "restriction"
	elemSchema             = "schema"
	elemSelector           = "selector"
	elemSequence           = "sequence"
	elemSimpleContent      = "simpleContent"
	elemSimpleType         = "simpleType"
	elemUnion              = "union"
	elemUnique             = "unique"
)

const elemExplicitTimezone = "explicitTimezone" // XSD 1.1: timezone-presence facet on temporal types

const (
	attrAbstract             = "abstract"
	attrAttributeFormDefault = "attributeFormDefault"
	attrBase                 = "base"
	attrBlock                = "block"
	attrBlockDefault         = "blockDefault"
	attrDefault              = "default"
	attrDefaultAttributes    = "defaultAttributes"      // XSD 1.1: schema-level default attribute group
	attrDefaultAttrsApply    = "defaultAttributesApply" // XSD 1.1: complexType default attribute toggle
	attrElementFormDefault   = "elementFormDefault"
	attrFinal                = "final"
	attrFinalDefault         = "finalDefault"
	attrFixed                = "fixed"
	attrForm                 = "form"
	attrInheritable          = "inheritable" // XSD 1.1: inheritable attribute
	attrItemType             = "itemType"
	attrMaxOccurs            = "maxOccurs"
	attrMemberTypes          = "memberTypes"
	attrMinOccurs            = "minOccurs"
	attrMode                 = "mode"           // XSD 1.1: xs:openContent mode
	attrAppliesToEmpty       = "appliesToEmpty" // XSD 1.1: xs:defaultOpenContent appliesToEmpty
	attrName                 = "name"
	attrNamespace            = "namespace"
	attrNotNamespace         = "notNamespace" // XSD 1.1: wildcard negated-namespace list
	attrNotQName             = "notQName"     // XSD 1.1: wildcard disallowed-names list
	attrNillable             = "nillable"
	attrNil                  = "nil"
	attrProcessContents      = "processContents"
	attrPublic               = "public" // xs:notation public identifier
	attrSystem               = "system" // xs:notation system identifier
	attrRefer                = "refer"
	attrRef                  = "ref"
	attrSchemaLocation       = "schemaLocation"
	attrNoNSSchemaLocation   = "noNamespaceSchemaLocation" // xsi: processor attribute
	attrSource               = "source"
	attrSubstitutionGroup    = "substitutionGroup"
	attrTargetNamespace      = "targetNamespace"
	attrTest                 = "test" // XSD 1.1: xs:assert/xs:assertion test expression
	attrType                 = "type"
	attrUse                  = "use"
	attrXPath                = "xpath"
	// attrXPathDefaultNamespace and attrXPathDefaultNS are two names for the SAME
	// XSD 1.1 xpathDefaultNamespace attribute (on xs:schema, xs:alternative,
	// xs:assert, xs:assertion, xs:selector, xs:field) supplying the default element
	// namespace for an XPath expression's unprefixed names. CTA code uses the first,
	// IDC code the second; both are kept to avoid churning every reference site.
	attrXPathDefaultNamespace = "xpathDefaultNamespace"
	attrXPathDefaultNS        = "xpathDefaultNamespace"
)

// XSD 1.1 xpathDefaultNamespace keyword values (in addition to a literal URI).
const (
	xpathDefaultNSTargetNamespace  = "##targetNamespace"
	xpathDefaultNSDefaultNamespace = "##defaultNamespace"
	xpathDefaultNSLocal            = "##local"
)

const (
	attrValExtension    = "extension"
	attrValLax          = "lax"
	attrValList         = "list"
	attrValOptional     = "optional"
	attrValProhibited   = "prohibited"
	attrValRequired     = "required"
	attrValRestriction  = "restriction"
	attrValSkip         = "skip"
	attrValStrict       = "strict"
	attrValSubstitution = "substitution"
	attrValUnbounded    = "unbounded"
	attrValUnion        = "union"
	attrValUnqualified  = "unqualified"
)

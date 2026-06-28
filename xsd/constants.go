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

const (
	elemAll            = "all"
	elemAlternative    = "alternative" // XSD 1.1: xs:alternative (conditional type assignment)
	elemAnnotation     = "annotation"
	elemAny            = "any"
	elemAnyAttribute   = "anyAttribute"
	elemAssert         = "assert"      // XSD 1.1: xs:assert on complex types
	elemAssertion      = "assertion"   // XSD 1.1: xs:assertion facet on simple types
	elemOpenContent    = "openContent" // XSD 1.1: xs:openContent on complex types
	elemAppinfo        = "appinfo"
	elemAttribute      = "attribute"
	elemAttributeGroup = "attributeGroup"
	elemChoice         = "choice"
	elemComplexContent = "complexContent"
	elemComplexType    = "complexType"
	elemDocumentation  = "documentation"
	elemElement        = "element"
	elemExtension      = "extension"
	elemField          = "field"
	elemGroup          = "group"
	elemImport         = "import"
	elemInclude        = "include"
	elemKey            = "key"
	elemKeyRef         = "keyref"
	elemList           = "list"
	elemRedefine       = "redefine"
	elemRestriction    = "restriction"
	elemSchema         = "schema"
	elemSelector       = "selector"
	elemSequence       = "sequence"
	elemSimpleContent  = "simpleContent"
	elemSimpleType     = "simpleType"
	elemUnion          = "union"
	elemUnique         = "unique"
)

const (
	attrAbstract             = "abstract"
	attrAttributeFormDefault = "attributeFormDefault"
	attrBase                 = "base"
	attrBlock                = "block"
	attrBlockDefault         = "blockDefault"
	attrDefault              = "default"
	attrElementFormDefault   = "elementFormDefault"
	attrFinal                = "final"
	attrFinalDefault         = "finalDefault"
	attrFixed                = "fixed"
	attrForm                 = "form"
	attrItemType             = "itemType"
	attrMaxOccurs            = "maxOccurs"
	attrMemberTypes          = "memberTypes"
	attrMinOccurs            = "minOccurs"
	attrMode                 = "mode" // XSD 1.1: xs:openContent mode
	attrName                 = "name"
	attrNamespace            = "namespace"
	attrNillable             = "nillable"
	attrNil                  = "nil"
	attrProcessContents      = "processContents"
	attrRefer                = "refer"
	attrRef                  = "ref"
	attrSchemaLocation       = "schemaLocation"
	attrSource               = "source"
	attrSubstitutionGroup    = "substitutionGroup"
	attrTargetNamespace      = "targetNamespace"
	attrTest                 = "test" // XSD 1.1: xs:assert/xs:assertion test expression
	attrType                 = "type"
	attrUse                  = "use"
	attrXPath                = "xpath"
	// attrXPathDefaultNamespace is the XSD 1.1 xpathDefaultNamespace attribute on
	// xs:schema/xs:assert/xs:assertion/xs:alternative, controlling the default
	// element namespace of an XPath expression (unprefixed element name tests).
	attrXPathDefaultNamespace = "xpathDefaultNamespace"
)

// xpathDefaultNamespace special token values (XSD 1.1 §3.1.2).
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
)

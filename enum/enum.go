// Package enum centralizes shared enumeration symbols used across packages.
// It exists so packages such as helium and sax can refer to the same typed
// constants for XML DTD declarations without redefining parallel enums.
// These values correspond to libxml2's C enumerations.
package enum

// AttributeType enumerates the XML attribute value types declared in a DTD
// (libxml2: xmlAttributeType).
type AttributeType int

const (
	AttrInvalid     AttributeType = iota // 0: not valid
	AttrCDATA                            // 1 (libxml2: XML_ATTRIBUTE_CDATA)
	AttrID                               // 2 (libxml2: XML_ATTRIBUTE_ID)
	AttrIDRef                            // 3 (libxml2: XML_ATTRIBUTE_IDREF)
	AttrIDRefs                           // 4 (libxml2: XML_ATTRIBUTE_IDREFS)
	AttrEntity                           // 5 (libxml2: XML_ATTRIBUTE_ENTITY)
	AttrEntities                         // 6 (libxml2: XML_ATTRIBUTE_ENTITIES)
	AttrNmtoken                          // 7 (libxml2: XML_ATTRIBUTE_NMTOKEN)
	AttrNmtokens                         // 8 (libxml2: XML_ATTRIBUTE_NMTOKENS)
	AttrEnumeration                      // 9 (libxml2: XML_ATTRIBUTE_ENUMERATION)
	AttrNotation                         // 10 (libxml2: XML_ATTRIBUTE_NOTATION)
)

// AttributeDefault enumerates the XML attribute default modes declared in a DTD
// (libxml2: xmlAttributeDefault).
type AttributeDefault int

const (
	AttrDefaultInvalid  AttributeDefault = iota // 0: not valid
	AttrDefaultNone                             // 1 (libxml2: XML_ATTRIBUTE_NONE)
	AttrDefaultRequired                         // 2 (libxml2: XML_ATTRIBUTE_REQUIRED)
	AttrDefaultImplied                          // 3 (libxml2: XML_ATTRIBUTE_IMPLIED)
	AttrDefaultFixed                            // 4 (libxml2: XML_ATTRIBUTE_FIXED)
)

// ElementType enumerates the element content types declared in a DTD
// (libxml2: xmlElementTypeVal).
type ElementType int

const (
	UndefinedElementType ElementType = iota // 0 (libxml2: XML_ELEMENT_TYPE_UNDEFINED)
	EmptyElementType                        // 1 (libxml2: XML_ELEMENT_TYPE_EMPTY)
	AnyElementType                          // 2 (libxml2: XML_ELEMENT_TYPE_ANY)
	MixedElementType                        // 3 (libxml2: XML_ELEMENT_TYPE_MIXED)
	ElementElementType                      // 4 (libxml2: XML_ELEMENT_TYPE_ELEMENT)
)

// EntityType enumerates the different kinds of XML entity
// (libxml2: xmlEntityType).
type EntityType int

const (
	InternalGeneralEntity         EntityType = iota + 1 // 1 (libxml2: XML_INTERNAL_GENERAL_ENTITY)
	ExternalGeneralParsedEntity                         // 2 (libxml2: XML_EXTERNAL_GENERAL_PARSED_ENTITY)
	ExternalGeneralUnparsedEntity                       // 3 (libxml2: XML_EXTERNAL_GENERAL_UNPARSED_ENTITY)
	InternalParameterEntity                             // 4 (libxml2: XML_INTERNAL_PARAMETER_ENTITY)
	ExternalParameterEntity                             // 5 (libxml2: XML_EXTERNAL_PARAMETER_ENTITY)
	InternalPredefinedEntity                            // 6 (libxml2: XML_INTERNAL_PREDEFINED_ENTITY)
)

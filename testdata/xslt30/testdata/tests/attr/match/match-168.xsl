<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:my="http://www.example.com/ns/match/var"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="my xs"
                input-type-annotations="strip"
                version="2.0">
<!-- Purpose: Test of template matching with attribute($name,xs:untypedAtomic) with typed attribute nodes from source file. 
				Stylesheet has @input-type-annotations=strip.-->

   <xslt:import-schema namespace="http://www.example.com/ns/match/var"
                       schema-location="variousTypesSchemaMatch.xsd"/>

   <xslt:template match="/">
	     	     <out>
         <xslt:apply-templates select="//@*"/>
      </out>
   </xslt:template>

   <xslt:template match="attribute(my:specialPart, xs:string)">
	     <xslt:element name="A">
		       <xslt:value-of select="."/>
	     </xslt:element>
         </xslt:template>

   <xslt:template match="attribute(my:listParts, my:myListType)">
	     <xslt:element name="B">
		       <xslt:value-of select="."/>
	     </xslt:element>
         </xslt:template>

   <xslt:template match="attribute(my:colors, xs:NMTOKENS)">
	     <xslt:element name="C">
		       <xslt:value-of select="."/>
	     </xslt:element>
         </xslt:template>

   <xslt:template match="attribute(my:union, my:partIntegerUnion)">
	     <xslt:element name="D">
		       <xslt:value-of select="."/>
	     </xslt:element>
         </xslt:template>

   <xslt:template match="attribute(my:specialPart, xs:untypedAtomic)">
	     <xslt:element name="E">
		       <xslt:value-of select="."/>
	     </xslt:element>
         </xslt:template>

   <xslt:template match="attribute(my:listParts, xs:untypedAtomic)">
	     <xslt:element name="F">
		       <xslt:value-of select="."/>
	     </xslt:element>
         </xslt:template>

   <xslt:template match="attribute(my:colors, xs:untypedAtomic)">
	     <xslt:element name="G">
		       <xslt:value-of select="."/>
	     </xslt:element>
         </xslt:template>

   <xslt:template match="attribute(my:union, xs:untypedAtomic)">
	     <xslt:element name="H">
		       <xslt:value-of select="."/>
	     </xslt:element>
         </xslt:template>

   <xslt:template match="attribute(*)">
</xslt:template>
</xslt:transform>

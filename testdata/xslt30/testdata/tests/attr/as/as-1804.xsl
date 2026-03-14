<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.example.com/ns/various" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test with global xsl:variable where @select has a typed value obtained at run time from attribute 
  				nodes and of the same type as the user-defined atomic type (derived by restriction) in @as. Verify 
  				the variable is of the type specified in @as.-->

   <xslt:import-schema namespace="http://www.example.com/ns/various"
      schema-location="variousTypesSchemaAs.xsd"/>

   <xslt:variable name="var1" select="/my:userNode/@my:specialPart" as="my:partNumberType"/>

   <xslt:variable name="var2" select="/my:userNode/@my_decimal"
      as="my:de1-decimal-enumeration-Inline"/>

   <xslt:template match="/my:userNode">
      <out>
         <var1>
            <xslt:value-of select="$var1 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var1 instance of my:partNumberType"/>
         </var1>
         <var2>
            <xslt:value-of select="$var2 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var2 instance of my:de1-decimal-enumeration-Inline"/>
         </var2>
      </out>
   </xslt:template>
</xslt:transform>

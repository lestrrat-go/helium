<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.example.com/ns/various" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test with global xsl:variable where value in @select is explicitly 
				constructed and of the same type as the user-defined atomic type 
				(derived by restriction) in @as. Verify the variable is of the type 
				specified in @as.-->

   <xslt:import-schema namespace="http://www.example.com/ns/various"
      schema-location="variousTypesSchemaAs.xsd"/>

   <xslt:variable name="var1" select="my:partNumberType('123-AB')" as="my:partNumberType"/>

   <xslt:variable name="var2" select="my:de1-decimal-enumeration-Inline(-1)"
      as="my:de1-decimal-enumeration-Inline"/>

   <xslt:template match="/doc">
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

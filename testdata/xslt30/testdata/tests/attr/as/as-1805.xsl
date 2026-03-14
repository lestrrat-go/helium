<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.example.com/ns/various" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test with local xsl:param without @select, where sequence constructor contains a typed value 
				obtained at run time from element or attribute nodes and is of the same type as the user-defined 
				atomic type (derived by restriction) in @as. Verify the parameter is of the type specified in @as.-->

   <xslt:import-schema namespace="http://www.example.com/ns/various"
      schema-location="variousTypesSchemaAs.xsd"/>

   <xslt:template match="/my:userNode">
      <xslt:param name="var1" select="/my:userNode/@my:specialPart" as="my:partNumberType"/>
      <xslt:param name="var2" select="/my:userNode/@my_decimal"
         as="my:de1-decimal-enumeration-Inline"/>
      <xslt:param name="var3" select="/my:userNode/my:simple-derived" as="my:partNumberType"/>
      <xslt:param name="var4" select="/my:userNode/my:de1-decimal-enumeration-inline"
         as="my:de1-decimal-enumeration-Inline"/>
      <out>
         <var1>
            <xslt:value-of select="$var1 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var1 instance of my:partNumberType"/>
         </var1>
         <var2>
            <xslt:value-of select="$var2 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var2 instance of my:de1-decimal-enumeration-Inline"/>
         </var2>
         <var3>
            <xslt:value-of select="$var3 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var3 instance of my:partNumberType"/>
         </var3>
         <var4>
            <xslt:value-of select="$var4 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var4 instance of my:de1-decimal-enumeration-Inline"/>
         </var4>
      </out>
   </xslt:template>
</xslt:transform>

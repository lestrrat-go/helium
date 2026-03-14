<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.example.com/ns/various" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test type of global xsl:variable where @select contains user-defined types derived by 
  				restriction from xs:decimal, @as="xs:float".(type promotion)-->

   <xslt:import-schema namespace="http://www.example.com/ns/various"
      schema-location="variousTypesSchemaAs.xsd"/>

   <xslt:variable name="var1" select="/my:userNode/my:de1-decimal-enumeration-inline" as="xs:float"/>

   <xslt:variable name="var2" select="/my:userNode/my:de2-decimal-maxExclusive-inline" as="xs:float"/>

   <xslt:variable name="var3" select="/my:userNode/my:de3-decimal-maxInclusive-minExclusive-inline"
      as="xs:float"/>

   <xslt:variable name="var4" select="/my:userNode/my:de4-decimal-minInclusive-pattern-inline"
      as="xs:float"/>

   <xslt:template match="/my:userNode">
      <out>
         <var1>
            <xslt:value-of select="$var1 instance of my:de1-decimal-enumeration-Inline"/>
            <xslt:value-of select="$var1 instance of xs:float"/>
         </var1>
         <var2>
            <xslt:value-of select="$var2 instance of my:de2-decimal-maxExclusive-Inline"/>
            <xslt:value-of select="$var2 instance of xs:float"/>
         </var2>
         <var3>
            <xslt:value-of
               select="$var3 instance of my:de3-decimal-maxInclusive-minExclusive-Inline"/>
            <xslt:value-of select="$var3 instance of xs:float"/>
         </var3>
         <var4>
            <xslt:value-of select="$var4 instance of my:de4-decimal-minInclusive-pattern-Inline"/>
            <xslt:value-of select="$var4 instance of xs:float"/>
         </var4>
      </out>
   </xslt:template>
</xslt:transform>

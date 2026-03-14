<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.example.com/ns/integer" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test type of global xsl:variable where @select contains a user-defined value derived by 
  				restriction from xs:int, @as="xs:int".(subtype substitution)-->

   <xslt:import-schema namespace="http://www.example.com/ns/integer"
      schema-location="integerSchemaAs.xsd"/>

   <xslt:variable name="var1" select="/my:integer-main/int2-int-maxExclusive" as="xs:int"/>

   <xslt:template match="/my:integer-main">
      <out>
         <var1>
            <xslt:value-of select="$var1 instance of my:int2-int-maxExclusive-Type"/>
            <xslt:value-of select="$var1 instance of xs:int"/>
         </var1>
      </out>
   </xslt:template>
</xslt:transform>

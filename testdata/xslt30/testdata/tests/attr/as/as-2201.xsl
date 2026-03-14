<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.example.com/ns/integer" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test type of global xsl:variable where @select contains a user-defined value derived by 
  				restriction from xs:integer, @as="xs:decimal".(subtype substitution)-->

   <xslt:import-schema namespace="http://www.example.com/ns/integer"
      schema-location="integerSchemaAs.xsd"/>

   <xslt:variable name="var1" select="/my:integer-main/i1-integer-enumeration" as="xs:decimal"/>

   <xslt:variable name="var2" select="/my:integer-main/i2-integer-maxExclusive-minExclusive"
      as="xs:decimal"/>

   <xslt:variable name="var3" select="/my:integer-main/i3-integer-maxInclusive-minInclusive"
      as="xs:decimal"/>

   <xslt:variable name="var4" select="/my:integer-main/i4-integer-pattern" as="xs:decimal"/>

   <xslt:variable name="var5" select="/my:integer-main/i5-integer-fractionDigits-totalDigits"
      as="xs:decimal"/>

   <xslt:variable name="var6" select="/my:integer-main/i6-integer-whiteSpace" as="xs:decimal"/>

   <xslt:template match="/my:integer-main">
      <out>
         <var1>
            <xslt:value-of select="$var1 instance of my:i1-integer-enumeration-Type"/>
            <xslt:value-of select="$var1 instance of xs:decimal"/>
         </var1>
         <var2>
            <xslt:value-of select="$var2 instance of my:i2-integer-maxExclusive-minExclusive-Type"/>
            <xslt:value-of select="$var2 instance of xs:decimal"/>
         </var2>
         <var3>
            <xslt:value-of select="$var3 instance of my:i3-integer-maxInclusive-minInclusive-Type"/>
            <xslt:value-of select="$var3 instance of xs:decimal"/>
         </var3>
         <var4>
            <xslt:value-of select="$var4 instance of my:i4-integer-pattern-Type"/>
            <xslt:value-of select="$var4 instance of xs:decimal"/>
         </var4>
         <var5>
            <xslt:value-of select="$var5 instance of my:i5-integer-fractionDigits-totalDigits-Type"/>
            <xslt:value-of select="$var5 instance of xs:decimal"/>
         </var5>
         <var6>
            <xslt:value-of select="$var6 instance of my:i6-integer-whiteSpace-Type"/>
            <xslt:value-of select="$var6 instance of xs:decimal"/>
         </var6>
      </out>
   </xslt:template>
</xslt:transform>

<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.example.com/ns/anyuri" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test type of global xsl:variable where @select contains user-defined types derived by restriction 
  				from xs:anyURI, @as="xs:string".(type promotion)-->

   <xslt:import-schema namespace="http://www.example.com/ns/anyuri"
      schema-location="anyUriSchemaAs.xsd"/>

   <xslt:variable name="var1" select="/my:anyUri-main/au1-anyURI-enumeration" as="xs:string"/>

   <xslt:variable name="var2" select="/my:anyUri-main/au2-anyURI-length" as="xs:string"/>

   <xslt:variable name="var3" select="/my:anyUri-main/au3-anyURI-minLength-pattern" as="xs:string"/>

   <xslt:variable name="var4" select="/my:anyUri-main/au4-anyURI-maxLength-whiteSpace"
      as="xs:string"/>

   <xslt:template match="/my:anyUri-main">
      <out>
         <var1>
            <xslt:value-of select="$var1 instance of my:au1-anyURI-enumeration-Type"/>
            <xslt:value-of select="$var1 instance of xs:string"/>
         </var1>
         <var2>
            <xslt:value-of select="$var2 instance of my:au2-anyURI-length-Type"/>
            <xslt:value-of select="$var2 instance of xs:string"/>
         </var2>
         <var3>
            <xslt:value-of select="$var3 instance of my:au3-anyURI-minLength-pattern-Type"/>
            <xslt:value-of select="$var3 instance of xs:string"/>
         </var3>
         <var4>
            <xslt:value-of select="$var4 instance of my:au4-anyURI-maxLength-whiteSpace-Type"/>
            <xslt:value-of select="$var4 instance of xs:string"/>
         </var4>
      </out>
   </xslt:template>
</xslt:transform>

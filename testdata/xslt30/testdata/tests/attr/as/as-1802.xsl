<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.example.com/ns/various" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test with global xsl:variable and @as="xs:anyAtomicType?", @select is:
				-empty-sequence, built-in derived atomic type, user-defined atomic type 
				derived by restriction -->

   <xslt:import-schema namespace="http://www.example.com/ns/various"
      schema-location="variousTypesSchemaAs.xsd"/>

   <xslt:variable name="var1" as="xs:anyAtomicType?">
      <xslt:sequence select="()"/>
   </xslt:variable>

   <xslt:variable name="var2" as="xs:anyAtomicType?">
      <xslt:value-of select="/my:userNode/my:elem-double"/>
   </xslt:variable>

   <xslt:variable name="var3" as="xs:anyAtomicType?">
      <xslt:value-of select="/my:userNode/my:simple-derived"/>
   </xslt:variable>

   <xslt:template match="/my:userNode">
      <out>
         <var1>
            <xslt:value-of select="$var1 instance of xs:anyAtomicType?"/>
            <xslt:value-of select="$var1 instance of xs:anyAtomicType"/>
         </var1>
         <var2>
            <xslt:value-of select="$var2 instance of xs:anyAtomicType?"/>
            <xslt:value-of select="$var2 instance of xs:anyAtomicType"/>
         </var2>
         <var3>
            <xslt:value-of select="$var3 instance of xs:anyAtomicType?"/>
            <xslt:value-of select="$var3 instance of xs:anyAtomicType"/>
         </var3>
      </out>
   </xslt:template>
</xslt:transform>

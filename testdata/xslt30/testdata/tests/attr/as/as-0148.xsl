<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.example.com/ns/various" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test of global xsl:variable with a sequence constructor of LREs and @as= user-defined atomic type 
  				(derived by restriction). Value of LRE is in the lexical space of the type in @as. Verify the variable 
  				is of the type specified in @as.-->

   <xslt:import-schema namespace="http://www.example.com/ns/various"
      schema-location="variousTypesSchemaAs.xsd"/>

   <xslt:variable name="var1" as="my:partNumberType">
      <item>123-AB</item>
   </xslt:variable>

   <xslt:variable name="var2" as="my:de1-decimal-enumeration-Inline">
      <item>0</item>
   </xslt:variable>

   <xslt:template match="/doc">
      <out>
         <var1>
            <xslt:value-of select="$var1"/>
            <xslt:text> * </xslt:text>
            <xslt:value-of select="$var1 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var1 instance of my:partNumberType"/>
         </var1>
         <var2>
            <xslt:value-of select="$var2"/>
            <xslt:text> * </xslt:text>
            <xslt:value-of select="$var2 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var2 instance of my:de1-decimal-enumeration-Inline"/>
         </var2>
      </out>
   </xslt:template>
</xslt:transform>

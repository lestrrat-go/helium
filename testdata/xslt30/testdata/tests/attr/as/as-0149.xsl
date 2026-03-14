<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.example.com/ns/various" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test with xsl:function where the value in the sequence 
  				constructor is explicitly constructed and is of the same 
  				type as the user-defined atomic type (derived by restriction) 
  				in @as. Verify the result of the function is of the type specified in @as.-->

   <xslt:import-schema namespace="http://www.example.com/ns/various"
      schema-location="variousTypesSchemaAs.xsd"/>

   <xslt:function name="my:func1" as="my:partNumberType">
      <xslt:value-of select="my:partNumberType('123-AB')"/>
   </xslt:function>

   <xslt:function name="my:func2" as="my:de1-decimal-enumeration-Inline">
      <xslt:value-of select="my:de1-decimal-enumeration-Inline(-1)"/>
   </xslt:function>

   <xslt:template match="/doc">
      <out>
         <func1>
            <xslt:value-of select="my:func1() instance of my:partNumberType"/>
         </func1>
         <func2>
            <xslt:value-of select="my:func2() instance of my:de1-decimal-enumeration-Inline"/>
         </func2>
      </out>
   </xslt:template>
</xslt:transform>

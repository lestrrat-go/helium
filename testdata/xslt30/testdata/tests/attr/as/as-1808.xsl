<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.example.com/ns/various" 
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="xs my" 
   version="2.0">
   <!-- Purpose: Test type of xsl:functions that returns user-defined types derived by restriction 
  				from xs:decimal or from xs:float, @as="xs:double".(type promotion)-->

   <xsl:import-schema namespace="http://www.example.com/ns/various"
      schema-location="variousTypesSchemaAs.xsd"/>

   <xsl:template match="/">
      <out>
         <func1>
            <xsl:if test="my:func1() != 12345678">
               <xsl:value-of select="my:func1() instance of my:de1-decimal-enumeration-Inline"/>
               <xsl:value-of select="my:func1() instance of xs:double"/>
            </xsl:if>
         </func1>
         <func2>
            <xsl:if test="my:func2() != 12345678">
               <xsl:value-of select="my:func2() instance of my:derived-float-Type"/>
               <xsl:value-of select="my:func2() instance of xs:double"/>
            </xsl:if>
         </func2>
      </out>
   </xsl:template>

   <xsl:function name="my:func1" as="xs:double">
      <xsl:value-of select="document('as-18.xml')/my:userNode/my:de1-decimal-enumeration-inline"/>
   </xsl:function>

   <xsl:function name="my:func2" as="xs:double">
      <xsl:value-of select="document('as-18.xml')/my:userNode/my:derived-float"/>
   </xsl:function>

</xsl:transform>

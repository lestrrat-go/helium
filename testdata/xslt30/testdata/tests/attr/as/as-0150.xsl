<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.example.com/ns/various" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test with xsl:function that returns an empty sequence and has @as=xs:NMTOKEN*. 
  				Verify the returned value if of type xs:NMTOKEN*.-->

   <xslt:import-schema namespace="http://www.example.com/ns/various"
      schema-location="variousTypesSchemaAs.xsd"/>

   <xslt:function name="my:func1" as="xs:NMTOKEN*">
      <xslt:sequence select="()"/>
   </xslt:function>

   <xslt:template match="/doc">
      <out>
         <func1>
            <xslt:value-of select="my:func1() instance of xs:NMTOKEN*"/>
            <xslt:value-of select="my:func1() instance of xs:NMTOKEN"/>
         </func1>
      </out>
   </xslt:template>
</xslt:transform>

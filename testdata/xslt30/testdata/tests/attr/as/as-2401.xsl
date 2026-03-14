<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.example.com/ns/various" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test type of global xsl:variable where @select contains a user-defined value derived by restriction 
  				from another user-defined type that is also specified in @as. (subtype substitution)-->

   <xslt:import-schema namespace="http://www.example.com/ns/various"
      schema-location="variousTypesSchemaAs.xsd"/>

   <xslt:variable name="var1" select="/my:userNode/my:derivedPart" as="my:partNumberType"/>

   <xslt:template match="/my:userNode">
      <out>
         <var1>
            <xslt:value-of select="$var1 instance of my:specialPartNumber"/>
            <xslt:value-of select="$var1 instance of my:partNumberType"/>
         </var1>
      </out>
   </xslt:template>
</xslt:transform>

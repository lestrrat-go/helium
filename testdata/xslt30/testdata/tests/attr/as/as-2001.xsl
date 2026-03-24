<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.example.com/ns/various" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test with xsl:template which contains an xs:untypedAtomic value obtained at run time and @as is a 
  				user-defined atomic type (derived by restriction). Verify no typing error is raised. -->

   <xslt:import-schema namespace="http://www.example.com/ns/various"
      schema-location="variousTypesSchemaAs.xsd"/>

   <xslt:template match="/my:userNode">
      <out>
         <temp1>
            <xslt:call-template name="temp1"/>
         </temp1>
         <temp2>
            <xslt:call-template name="temp2"/>
         </temp2>
      </out>
   </xslt:template>

   <xslt:template name="temp1" as="my:partNumberType">
      <xslt:value-of select="/my:userNode/my:simple-derived"/>
   </xslt:template>

   <xslt:template name="temp2" as="my:de1-decimal-enumeration-Inline">
      <xslt:value-of select="/my:userNode/my:de1-decimal-enumeration-inline"/>
   </xslt:template>
</xslt:transform>

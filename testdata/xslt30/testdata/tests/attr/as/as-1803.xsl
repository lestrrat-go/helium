<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.example.com/ns/various" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test with xsl:template which returns a mixed sequence of derived 
  				built-in or user-defined atomic values and @as="xs:anyAtomicType+". -->

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

   <xslt:template name="temp1" as="xs:anyAtomicType+">
      <xslt:value-of select="/my:userNode/my:elem-anyURI"/>
      <xslt:text> * </xslt:text>
      <xslt:value-of select="/my:userNode/my:elem-duration"/>
      <xslt:text> * </xslt:text>
      <xslt:value-of select="abs(/my:userNode/my:elem-double)"/>
      <xslt:text> * </xslt:text>
      <xslt:value-of select="/my:userNode/my:elem-nonPositiveInteger"/>
      <xslt:text> * </xslt:text>
      <xslt:value-of select="/my:userNode/my:elem-NMTOKEN"/>
   </xslt:template>

   <xslt:template name="temp2" as="xs:anyAtomicType+">
      <xslt:value-of select="/my:userNode/my:simple-derived"/>
      <xslt:text> * </xslt:text>
      <xslt:value-of select="/my:userNode/my:de1-decimal-enumeration-inline"/>
      <xslt:text> * </xslt:text>
      <xslt:value-of select="/my:userNode/my:simpleUserElem"/>
      <xslt:text> * </xslt:text>
      <xslt:value-of select="/my:userNode/my:de4-decimal-minInclusive-pattern-inline"/>
   </xslt:template>
</xslt:transform>

<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.example.com/ns/various" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test with xsl:template which contains an explicitly constructed typed value 
  				and @as="xs:anyAtomicType". Types tested: - derived built-in atomic type,
				user-defined atomic type-->

   <xslt:import-schema namespace="http://www.example.com/ns/various"
      schema-location="variousTypesSchemaAs.xsd"/>

   <xslt:template match="/doc">
      <out>
         <temp1>
            <xslt:call-template name="temp1"/>
         </temp1>
         <temp2>
            <xslt:call-template name="temp2"/>
         </temp2>
         <temp3>
            <xslt:call-template name="temp3"/>
         </temp3>
         <temp4>
            <xslt:call-template name="temp4"/>
         </temp4>
      </out>
   </xslt:template>

   <xslt:template name="temp1" as="xs:anyAtomicType">
      <xslt:value-of select="xs:duration('P1Y2M3DT10H30M23S')"/>
   </xslt:template>

   <xslt:template name="temp2" as="xs:anyAtomicType">
      <xslt:value-of select="xs:gYear('2005')"/>
   </xslt:template>

   <xslt:template name="temp3" as="xs:anyAtomicType">
      <xslt:value-of select="my:de1-decimal-enumeration-Inline(1.0001)"/>
   </xslt:template>

   <xslt:template name="temp4" as="xs:anyAtomicType">
      <xslt:value-of select="my:partNumberType('000-TM')"/>
   </xslt:template>
</xslt:transform>

<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.example.com/ns/various" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test of xsl:template with  @as="element(*, user-defined-atomic-type) ?". 
  				The sequence constructor contains: -nothing, an empty sequence, a typed 
  				element node from input file, xsl:element with @type-->

   <xslt:import-schema namespace="http://www.example.com/ns/various"
      schema-location="variousTypesSchemaAs.xsd"/>

   <xslt:template match="/">
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

   <xslt:template name="temp1" as="element(*,my:partNumberType)?"> </xslt:template>

   <xslt:template name="temp2" as="element(*,my:partNumberType)?">
      <xslt:sequence select="()"/>
   </xslt:template>

   <xslt:template name="temp3" as="element(*,my:partNumberType)?">
      <xslt:copy-of select="my:userNode/my:simpleUserElem[1]" validation="strict"/>
   </xslt:template>

   <xslt:template name="temp4" as="element(*,my:partNumberType)?">
      <xslt:element name="item" type="my:partNumberType">000-TM</xslt:element>
   </xslt:template>
</xslt:transform>

<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.example.com/ns/various" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test of xsl:template which contains a typed element node from input file, 
  				or explicitly created xsl:element or LRE with xsl:type, @as="element(*, user-defined-complex-type)". 
  				Content of complex type is: -empty , element only, mixed -->

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
         <temp2>
            <xslt:call-template name="temp3"/>
         </temp2>
      </out>
   </xslt:template>

   <xslt:template name="temp1" as="element(*, my:onlyAttributesType)">
      <xslt:copy-of select="my:onlyAttributes" validation="strict"/>
   </xslt:template>

   <xslt:template name="temp2" as="element(*, my:personType)">
      <my:complexUserElem xslt:type="my:personType" age="75">
         <firstName>Hello</firstName>
         <familyName>There</familyName>
      </my:complexUserElem>
   </xslt:template>

   <xslt:template name="temp3" as="element(*, my:mixedPersonType)">
      <xslt:element name="my:complexMixedUserElem" validation="strict">
         <xslt:attribute name="age">44</xslt:attribute>
         <xslt:element name="description">Mr <xslt:element name="name">Peter</xslt:element> has brown hair </xslt:element>
         <xslt:element name="company">He works at a <xslt:element name="name">Notorios</xslt:element> company </xslt:element>
      </xslt:element>

   </xslt:template>
</xslt:transform>

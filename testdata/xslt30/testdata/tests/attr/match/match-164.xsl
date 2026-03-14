<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:my="http://www.example.com/ns/match/var"
   xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="my xs" version="2.0">
   <!-- Purpose: Test of template matching with attribute( * ,$type) where $type has a prefix.-->

   <xslt:import-schema namespace="http://www.example.com/ns/match/var"
      schema-location="variousTypesSchemaMatch.xsd"/>

   <xslt:template match="/">
      <out>
         <xslt:apply-templates select="//@*"/>
      </out>
   </xslt:template>

   <xslt:template match="attribute(*, my:partNumberType)" priority="5">
      <xslt:element name="A">
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="attribute(*, my:myListType)" priority="4.5">
      <xslt:element name="B">
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="attribute(*, xs:NMTOKENS)" priority="4.5">
      <xslt:element name="C">
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="attribute(*, my:partIntegerUnion)" priority="3">
      <xslt:element name="D">
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="attribute(*)" priority="4"> </xslt:template>
</xslt:transform>

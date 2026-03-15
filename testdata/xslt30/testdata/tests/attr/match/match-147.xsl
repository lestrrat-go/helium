<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:my="http://www.example.com/ns/match/var"
   xmlns:pre="http://www.example.com/ns/match/var" xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="my xs pre"
   xpath-default-namespace="http://www.example.com/ns/match/var" version="2.0">
   <!-- Purpose: Test of template matching with element($name ,$type) where $name has no prefix. 
  				Show effect of @xpath-default-namespace on $name.-->

   <xslt:import-schema namespace="http://www.example.com/ns/match/var"
      schema-location="variousTypesSchemaMatch.xsd"/>

   <xslt:template match="element(userNode, my:userType)">
      <out>
         <xslt:apply-templates select="*"/>
      </out>
   </xslt:template>

   <xslt:template match="element(simpleBuiltin, xs:anyURI)">
      <xslt:element name="A{position()}">
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="element(simpleUserElem, pre:partNumberType)">
      <xslt:element name="B{position()}">
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="element(simpleUserList, my:myListType)">
      <xslt:element name="C{position()}">
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="element(simpleUserUnion, pre:partIntegerUnion)">
      <xslt:element name="D{position()}">
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="element(complexSimpleContentElem, xs:decimal)">
      <xslt:element name="E{position()}">
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="element(complexMixedUserElem, my:mixedPersonType)">
      <xslt:element name="F{position()}">
         <xslt:value-of select="./*[1]"/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="element(*)"> </xslt:template>
</xslt:transform>

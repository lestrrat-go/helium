<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:my="http://www.example.com/ns/match/var"
   xmlns:pre="http://www.example.com/ns/match/var" xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="my xs pre"
   version="3.0">
   
   <!-- derived from match-158 -->
   <!-- Purpose: Test of xsl:mode typed="strict" with union patterns.-->

   <xslt:import-schema namespace="http://www.example.com/ns/match/var"
      schema-location="variousTypesSchemaMatch.xsd"/>
      
   <xslt:mode typed="strict"/>   

   <xslt:template match="my:userNode">
      <out>
         <xslt:apply-templates select="*"/>
      </out>
   </xslt:template>

   <xslt:template
      match="my:simpleBuiltin | my:simpleUserElem | my:simpleUserList | //my:simpleUserUnion">
      <xslt:element name="A{position()}">
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="element(*)"> </xslt:template>
</xslt:transform>

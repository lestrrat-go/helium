<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:my="http://www.example.com/ns/match/var"
   xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="my xs" version="2.0">
   <!-- Purpose: Test of template matching with "/attribute($name,$type)" with an explicitly created and typed 
  				attribute node with a typed parent. Does not match.-->

   <xslt:import-schema namespace="http://www.example.com/ns/match/var"
      schema-location="variousTypesSchemaMatch.xsd"/>

   <xslt:variable name="var">
      <xslt:element name="my:userNode" type="my:userType">
         <xslt:attribute name="my:specialPart" type="my:partNumberType">000-TM</xslt:attribute>
      </xslt:element>
   </xslt:variable>

   <xslt:template match="/doc">
      <out>
         <xslt:apply-templates select="$var"/>
      </out>
   </xslt:template>

   <xslt:template match="/attribute(my:specialPart, xs:string)">
      <xslt:element name="A">
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="element(*)">
      <xslt:apply-templates select="@*"/>
   </xslt:template>

   <xslt:template match="attribute(*)">
      <E>
         <xslt:value-of select="."/>
         <xslt:text>correct</xslt:text>
      </E>
   </xslt:template>
</xslt:transform>

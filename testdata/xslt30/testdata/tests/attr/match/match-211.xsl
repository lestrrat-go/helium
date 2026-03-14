<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:my="http://www.example.com/ns/match/attributes"
   xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="my xs" version="2.0">
   <!-- Purpose: Test of template matching with schema-attribute($name) for top-level attribute declarations of a 
  				user-defined union type. Show all cases where a node will match the pattern-->

   <xslt:import-schema namespace="http://www.example.com/ns/match/attributes"
      schema-location="match-attributes.xsd"/>

   <xslt:template match="/my:people">
      <out>
         <xslt:apply-templates select="//@*"/>
      </out>
   </xslt:template>

   <xslt:template match="attribute(my:string-int-union, xs:string)">
      <xslt:element name="A">
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="attribute(my:nationality-decimal-union, my:nationality-decimal-type)">
      <xslt:element name="B">
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="attribute(my:string-int-derived, my:string-int-type)">
      <xslt:element name="C">
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="attribute(*)"> </xslt:template>
</xslt:transform>

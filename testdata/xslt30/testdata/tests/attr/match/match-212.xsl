<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:my="http://www.example.com/ns/match/id-idref-notation"
   xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="my xs" version="2.0">
   <!-- Purpose: Test of template matching with function id(). Source file has attributes and elements of type xs:ID.-->

   <xslt:import-schema namespace="http://www.example.com/ns/match/id-idref-notation"
      schema-location="match-id-idref-notation.xsd"/>

   <xslt:template match="/">
      <out>
         <xslt:apply-templates select="//@*"/>
         <xslt:apply-templates select="//*"/>
      </out>
   </xslt:template>

   <xslt:template match="id('a1')">
      <xslt:element name="A">
         <xslt:value-of select="./name()"/>
         <xslt:text> * </xslt:text>
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="id('a2')">
      <xslt:element name="B">
         <xslt:value-of select="./name()"/>
         <xslt:text> * </xslt:text>
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="id('unique')">
      <xslt:element name="C">
         <xslt:value-of select="./name()"/>
         <xslt:text> * </xslt:text>
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="id('a3')">
      <xslt:element name="D">
         <xslt:value-of select="./name()"/>
         <xslt:text> * </xslt:text>
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="attribute(*)"> </xslt:template>

   <xslt:template match="element(*)"> </xslt:template>
</xslt:transform>

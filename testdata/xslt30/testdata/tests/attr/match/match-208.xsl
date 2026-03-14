<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:my="http://www.example.com/ns/match/id-idref-notation"
   xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="my xs" version="2.0">
   <!-- Purpose: Test of template matching with attribute($name,$type) for derived 
  				built-in atomic types including xs:ID, xs:IDREF, xs:NOTATION, xs:ENTITY.-->

   <xslt:import-schema namespace="http://www.example.com/ns/match/id-idref-notation"
      schema-location="match-id-idref-notation.xsd"/>

   <xslt:template match="/">
      <out>
         <xslt:apply-templates select="//@*"/>
      </out>
   </xslt:template>

   <xslt:template match="attribute(my:id, xs:ID)">
      <xslt:element name="A">
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="attribute(my:ref, xs:IDREF)">
      <xslt:element name="B">
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="attribute(my:filetype, xs:NOTATION)">
      <xslt:element name="C">
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="attribute(my:entitytype, xs:ENTITY)">
      <xslt:element name="D">
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="attribute(my:entities, xs:ENTITIES)">
      <xslt:element name="E">
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="attribute(my:attr-token, xs:token)">
      <xslt:element name="F">
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="attribute(my:attr-language, xs:language)">
      <xslt:element name="G">
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="attribute(my:attr-positiveInteger, xs:positiveInteger)">
      <xslt:element name="H">
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="attribute(my:refs, xs:IDREFS)">
      <xslt:element name="I">
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="attribute(*)"> </xslt:template>
</xslt:transform>

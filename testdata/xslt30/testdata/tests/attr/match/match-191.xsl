<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:my="http://www.example.com/ns/match/attributes"
   xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="my xs" version="2.0">
   <!-- Purpose:Test of template matching with schema-attribute($name) with an explicitly created and typed attribute node without a parent.-->

   <xslt:import-schema namespace="http://www.example.com/ns/match/attributes"
      schema-location="match-attributes.xsd"/>

   <xslt:variable name="var1" as="attribute()">
      <xslt:attribute name="my:nationality" type="xs:string">american</xslt:attribute>
   </xslt:variable>

   <xslt:variable name="var2" as="attribute()">
      <xslt:attribute name="my:nationality" type="my:nameNationality">latin</xslt:attribute>
   </xslt:variable>

   <xslt:template match="/">
      <out>
         <xslt:apply-templates select="$var1"/>
         <xslt:apply-templates select="$var2"/>
      </out>
   </xslt:template>

   <xslt:template match="schema-attribute(my:nationality)">
      <xslt:element name="A">
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="attribute(*)"> </xslt:template>
</xslt:transform>

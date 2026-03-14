<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:my="http://www.example.com/ns/match/var"
   xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="my xs" version="2.0">
   <!-- Purpose: Test of template matching with schema-attribute($name) with an explicitly 
  				created and typed attribute node with a typed parent.-->

   <xslt:import-schema namespace="http://www.example.com/ns/match/var"
      schema-location="variousTypesSchemaMatch.xsd"/>

   <xslt:template match="/">
      <out>
         <xslt:apply-templates select="$v"/>
      </out>
   </xslt:template>

   <xslt:template match="schema-attribute(my:specialPart)">
      <xslt:element name="A">
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="element(my:userNode)">
      <xslt:apply-templates select="@*"/>
   </xslt:template>

   <xslt:template match="attribute(*)">
      <E>
         <xslt:value-of select="."/>
         <xslt:text>wrong</xslt:text>
      </E>
   </xslt:template>
   
   <xslt:variable name="v" as="element()">
      <my:userNode xslt:validation="strict">
         <xslt:attribute name="my:specialPart">000-TM</xslt:attribute>
      </my:userNode>
   </xslt:variable>
   
</xslt:transform>

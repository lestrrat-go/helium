<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:my="http://www.example.com/ns/match/var"
   xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="my xs" version="2.0">
   <!-- Purpose: Test of template matching with schema-attribute($name) with an explicitly constructed 
  				attribute node without a parent and validated with @validation=strict. -->
   
   <!-- Copied from match-193, but using a local variable rather than a global variable. Saxon bug 5732 -->

   <xslt:import-schema namespace="http://www.example.com/ns/match/var"
      schema-location="variousTypesSchemaMatch.xsd"/>

   <xslt:template match="/">
      <out>
         <xslt:variable name="var" as="attribute()">
            <xslt:attribute name="my:specialPart" validation="strict">000-TM</xslt:attribute>
         </xslt:variable>
         <xslt:apply-templates select="$var"/>
      </out>
   </xslt:template>

   <xslt:template match="schema-attribute(my:specialPart)">
      <xslt:element name="A">
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="attribute(*)"> </xslt:template>
</xslt:transform>

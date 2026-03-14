<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:my="http://www.example.com/ns/match/var"
   xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="my xs" version="2.0">
   <!-- Purpose: Test of template matching with attribute($name ,$type) where $name has no prefix. 
  				Show there is no effect of @xpath-default-namespace on $name.-->

   <xslt:import-schema namespace="http://www.example.com/ns/match/var"
      schema-location="variousTypesSchemaMatch.xsd"/>

   <xslt:template match="/">
      <out>
         <xslt:apply-templates select="//@*"/>
      </out>
   </xslt:template>

   <xslt:template match="attribute(specialPart, xs:string)"
      xpath-default-namespace="http://www.example.com/ns/match/var">
      <xslt:element name="A">
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>

   <xslt:template match="attribute(*)"> </xslt:template>
</xslt:transform>

<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:my="http://www.example.com/ns/match/var"
   xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="my xs" input-type-annotations="strip" version="2.0">
   <!-- Purpose: Test of template matching with element($name,xs:untyped) with typed nodes from source file. 
  				Stylesheet has @input-type-annotations=strip.-->

   <xslt:import-schema namespace="http://www.example.com/ns/match/var"
      schema-location="variousTypesSchemaMatch.xsd"/>

   <xslt:template match="/my:userNode">
      <out>
         <xslt:apply-templates select="*"/>
      </out>
   </xslt:template>

   <xslt:template match="element(*, xs:untyped)">
      <xslt:element name="E{position()}">
         <xslt:value-of select="."/>
      </xslt:element>
   </xslt:template>
</xslt:transform>

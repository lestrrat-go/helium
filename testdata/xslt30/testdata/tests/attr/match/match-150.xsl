<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:my="http://www.example.com/ns/match/var"
   xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="my xs" version="2.0">
   <!-- Purpose: Test of template matching with element($name,$type) as the last step in a path expression.-->

   <xslt:import-schema namespace="http://www.example.com/ns/match/var"
      schema-location="variousTypesSchemaMatch.xsd"/>

   <xslt:template match="/my:userNode">
      <out>
         <xslt:apply-templates/>
      </out>
   </xslt:template>

   <xslt:template match="my:userNode/my:complexUserElem/element(firstName, xs:string)">
      <E>
         <xslt:value-of select="."/>
      </E>
   </xslt:template>

   <xslt:template
      match="element(my:userNode, my:userType)/element(my:complexUserElem, my:personType)">
      <xslt:apply-templates/>
   </xslt:template>

   <xslt:template match="element()"> </xslt:template>
</xslt:transform>

<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.example.com/ns/match/subgroup"
   xmlns:pre="http://www.example.com/ns/match/subgroup"
   xmlns:t="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="my pre" version="2.0">
   <!-- Purpose: Test of template matching with schema-element($name) for top-level element declarations of a complex type. Show all cases where a node will match the pattern -->

   <t:import-schema namespace="http://www.example.com/ns/match/subgroup"
      schema-location="subgroupTypeSchemaMatch.xsd"/>

   <t:template match="schema-element(my:doc)">
      <out>
         <t:apply-templates select="*"/>
      </out>
   </t:template>

   <t:template match="schema-element(pre:elem-anyURI)">
      <t:element name="A{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="element(*)">
      <t:apply-templates select="*"/>
   </t:template>
</t:transform>

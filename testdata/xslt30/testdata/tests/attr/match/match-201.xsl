<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.example.com/ns/match/subgroup"
   xmlns:pre="http://www.example.com/ns/match/subgroup"
   xmlns:t="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="my pre" version="2.0">
   <!-- Purpose: Test of template matching with document-node(element($name,$type)) for a 
  				complex type which is the base type of the document element.-->

   <t:import-schema namespace="http://www.example.com/ns/match/subgroup"
      schema-location="subgroupTypeSchemaMatch.xsd"/>

   <t:template match="document-node(element(*, my:docType))">
      <out>
         <t:value-of select=". instance of document-node()"/>
         <t:text> * </t:text>
         <t:value-of select="./*[1]/name()"/>
      </out>
   </t:template>

   <t:template match="element(*)"> </t:template>
</t:transform>

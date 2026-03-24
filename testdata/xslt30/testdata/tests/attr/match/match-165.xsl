<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test of template matching with attribute( * ,$type) where $type has no prefix. 
  				Show effect of @xpath-default-namespace on $type.-->

   <t:import-schema namespace="http://www.example.com/ns/match/var"
      schema-location="variousTypesSchemaMatch.xsd"/>

   <t:template match="/">
      <out>
         <t:apply-templates select="//@*"/>
      </out>
   </t:template>

   <t:template match="attribute(*, partNumberType)"
      xpath-default-namespace="http://www.example.com/ns/match/var">
      <t:element name="A">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="attribute(*)"> </t:template>
</t:transform>

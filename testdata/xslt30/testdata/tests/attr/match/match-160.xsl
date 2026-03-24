<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test of template matching with document-node(element( * ,$type)) where $type has no prefix. 
  				Show effect of @xpath-default-namespace on $type.-->

   <t:import-schema namespace="http://www.example.com/ns/match/var"
      schema-location="variousTypesSchemaMatch.xsd"/>

   <t:template match="document-node(element(*, userType))"
      xpath-default-namespace="http://www.example.com/ns/match/var">
      <out>
         <t:value-of select=". instance of document-node()"/>
         <t:text> * </t:text>
         <t:value-of select="./*[1]/name()"/>
      </out>
   </t:template>

   <t:template match="element(*)"> </t:template>
</t:transform>

<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test of template matching with document-node(schema-element($name)) 
  				where $name has no prefix and it belongs to no namespace.-->

   <t:import-schema schema-location="match-builtin.xsd"/>

   <t:template match="document-node(schema-element(doc-match))">
      <out>
         <t:value-of select=". instance of document-node()"/>
         <t:text> * </t:text>
         <t:value-of select="./*[1]/name()"/>
      </out>
   </t:template>

   <t:template match="element(*)"> </t:template>
</t:transform>

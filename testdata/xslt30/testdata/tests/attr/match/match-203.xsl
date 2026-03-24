<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test of template matching with attribute( * ,$type) 
  				where $type has no prefix and it belongs to no namespace.-->

   <t:import-schema schema-location="match-builtin.xsd"/>

   <t:template match="/">
      <out>
         <t:apply-templates select="//@*"/>
      </out>
   </t:template>

   <t:template match="attribute(*, de1-decimal-enumeration-Match)">
      <t:value-of select="."/>
   </t:template>

   <t:template match="attribute(*)"> </t:template>
</t:transform>

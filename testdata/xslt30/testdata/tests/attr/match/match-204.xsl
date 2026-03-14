<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test of template matching with schema-attribute($name) where $name 
  				has no prefix and it belongs to no namespace.-->

   <t:import-schema schema-location="match-builtin.xsd"/>

   <t:template match="/">
      <out>
         <t:apply-templates select="/*/*/@*"/>
      </out>
   </t:template>

   <t:template match="schema-attribute(attr-de1-decimal-enumeration-match)">
      <t:element name="A">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="schema-attribute(attr-base64Binary-match)">
      <t:element name="B">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="attribute(*)">
      <C>
         <t:value-of select="."/>
      </C>
   </t:template>
</t:transform>

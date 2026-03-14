<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test of template matching with schema-element($name) 
  				where $name has no prefix and it belongs to no namespace.-->

   <t:import-schema schema-location="match-builtin.xsd"/>

   <t:template match="schema-element(doc-match)">
      <out>
         <t:apply-templates select="*"/>
      </out>
   </t:template>

   <t:template match="schema-element(de1-decimal-enumeration-match)">
      <t:element name="E{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="schema-element(de2-decimal-maxExclusive-match)">
      <t:element name="E{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="schema-element(de3-decimal-maxInclusive-minExclusive-match)">
      <t:element name="E{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="schema-element(de4-decimal-minInclusive-pattern-match)">
      <t:element name="E{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="schema-element(de5-decimal-fractionDigits-totalDigits-match)">
      <t:element name="E{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="schema-element(de6-decimal-whiteSpace-match)">
      <t:element name="E{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="element(*)"> </t:template>
</t:transform>

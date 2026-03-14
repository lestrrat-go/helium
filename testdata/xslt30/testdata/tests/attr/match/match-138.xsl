<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test of template matching with element($name ,$type) where $name has no prefix.-->

   <t:import-schema schema-location="match-builtin.xsd"/>

   <t:template match="element(doc-match, docType-match)">
      <out>
         <t:apply-templates select="*"/>
      </out>
   </t:template>

   <t:template match="element(de1-decimal-enumeration-match, de1-decimal-enumeration-Match)">
      <t:element name="E{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="element(de2-decimal-maxExclusive-match, de2-decimal-maxExclusive-Match)">
      <t:element name="E{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template
      match="element(de3-decimal-maxInclusive-minExclusive-match, de3-decimal-maxInclusive-minExclusive-Match)">
      <t:element name="E{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template
      match="element(de4-decimal-minInclusive-pattern-match, de4-decimal-minInclusive-pattern-Match)">
      <t:element name="E{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template
      match="element(de5-decimal-fractionDigits-totalDigits-match, de5-decimal-fractionDigits-totalDigits-Match)">
      <t:element name="E{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="element(de6-decimal-whiteSpace-match, de6-decimal-whiteSpace-Match)">
      <t:element name="E{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="element(*)"> </t:template>
</t:transform>

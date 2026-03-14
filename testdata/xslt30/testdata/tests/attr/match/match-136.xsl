<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test of template matching with element( * ,$type) where 
  				$type has no prefix and it belongs to no namespace.-->

   <t:import-schema schema-location="match-builtin.xsd"/>

   <t:template match="element(*, docType-match)">
      <out>
         <t:apply-templates select="*"/>
      </out>
   </t:template>

   <t:template match="element(*, de1-decimal-enumeration-Match)">
      <t:element name="E{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="element(*, de2-decimal-maxExclusive-Match)">
      <t:element name="E{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="element(*, de3-decimal-maxInclusive-minExclusive-Match)">
      <t:element name="E{position()}">
         <t:value-of select="."/>
      </t:element>

   </t:template>

   <t:template match="element(*, de4-decimal-minInclusive-pattern-Match)">
      <t:element name="E{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="element(*, de5-decimal-fractionDigits-totalDigits-Match)">
      <t:element name="E{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="element(*, de6-decimal-whiteSpace-Match)">
      <t:element name="E{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="element(*)"> </t:template>
</t:transform>

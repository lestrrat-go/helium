<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
   <!-- Purpose: Test of template matching with implicit schema-element($name) 
  	    Derived from match-140 -->

   <t:import-schema schema-location="match-builtin.xsd"/>
   
   <t:mode typed="strict"/>

   <t:template match="doc-match">
      <out>
         <t:apply-templates select="*"/>
         <t:variable name="temp" as="element()">
           <de1-decimal-enumeration-match t:type="xs:string">DECOY</de1-decimal-enumeration-match>
         </t:variable>
         <t:apply-templates select="$temp"/>  
      </out>
   </t:template>
   
   <t:template match="*:de1-decimal-enumeration-match">
     <!-- matches an untyped element -->
     <ok/>
   </t:template>

   <t:template match="de1-decimal-enumeration-match">
      <t:element name="E{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="de2-decimal-maxExclusive-match">
      <t:element name="E{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="de3-decimal-maxInclusive-minExclusive-match">
      <t:element name="E{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="de4-decimal-minInclusive-pattern-match">
      <t:element name="E{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="de5-decimal-fractionDigits-totalDigits-match">
      <t:element name="E{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="de6-decimal-whiteSpace-match">
      <t:element name="E{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="element(*)"> </t:template>
</t:transform>

<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform"
   xpath-default-namespace="http://www.example.com/ns/match/var" version="2.0">
   <!-- Purpose: Test of template matching with element( * ,$type) where $type has no prefix. 
  				Show effect of @xpath-default-namespace on $type.-->

   <t:import-schema namespace="http://www.example.com/ns/match/var"
      schema-location="variousTypesSchemaMatch.xsd"/>

   <t:template match="element(*, userType)">
      <out>
         <t:apply-templates select="*"/>
      </out>
   </t:template>

   <t:template match="element(*, anyURI)" xpath-default-namespace="http://www.w3.org/2001/XMLSchema">
      <t:element name="A{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="element(*, partNumberType)" priority="4">
      <t:element name="B{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="element(*, myListType)">
      <t:element name="C{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="element(*, partIntegerUnion)" priority="3">
      <t:element name="D{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="element(*, decimal)"
      xpath-default-namespace="http://www.w3.org/2001/XMLSchema">
      <t:element name="E{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="element(*, mixedPersonType)">
      <t:element name="F{position()}">
         <t:value-of select="./*[1]"/>
      </t:element>
   </t:template>

   <t:template match="element(*)"> </t:template>
</t:transform>

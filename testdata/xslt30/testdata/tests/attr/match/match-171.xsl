<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.example.com/ns/match/var"
   xmlns:xs="http://www.w3.org/2001/XMLSchema" 
   xmlns:t="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="my xs" version="2.0">
   <!-- Purpose: Test of template matching with schema-attribute($name) Following an axis. -->

   <t:import-schema namespace="http://www.example.com/ns/match/var"
      schema-location="variousTypesSchemaMatch.xsd"/>

   <t:template match="/">
      <out>
         <t:apply-templates select="//@*"/>
      </out>
   </t:template>

   <t:template match="child::schema-attribute(my:listParts)">
      <t:element name="B">
         <t:value-of select="."/>
      </t:element>
   </t:template>
   
   <t:template match="attribute::schema-attribute(my:specialPart)">
      <t:element name="A">
         <t:value-of select="."/>
      </t:element>
   </t:template>   

   <t:template match="attribute::schema-attribute(my:colors)">
      <t:element name="C">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="child::schema-attribute(my:union)">
      <t:element name="D">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="attribute::attribute(*)"/> 
</t:transform>

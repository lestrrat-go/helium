<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform"
   xpath-default-namespace="http://www.example.com/ns/match/var" version="2.0">
   <!-- Purpose: Test of template matching with schema-element($name) where $name has no prefix. 
  				Show effect of @xpath-default-namespace on $name.-->

   <t:import-schema namespace="http://www.example.com/ns/match/var"
      schema-location="variousTypesSchemaMatch.xsd"/>

   <t:template match="schema-element(userNode)">
      <out>
         <t:apply-templates select="*"/>
      </out>
   </t:template>

   <t:template match="schema-element(simpleBuiltin)">
      <t:element name="A{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="schema-element(simpleUserElem)">
      <t:element name="B{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="schema-element(simpleUserList)">
      <t:element name="C{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="schema-element(simpleUserUnion)">
      <t:element name="D{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="schema-element(complexSimpleContentElem)">
      <t:element name="E{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="schema-element(complexMixedUserElem)">
      <t:element name="F{position()}">
         <t:value-of select="./*[1]"/>
      </t:element>
   </t:template>

   <t:template match="element(*)"> </t:template>
</t:transform>

<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.example.com/ns/match/var"
   xmlns:t="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="my" version="2.0">
   <!-- Purpose: Test of template matching with element($name,$type) and element( * ,$type) with a predicate.-->

   <t:import-schema namespace="http://www.example.com/ns/match/var"
      schema-location="variousTypesSchemaMatch.xsd"/>

   <t:template match="/my:userNode">
      <out>
         <t:apply-templates select="*"/>
      </out>
   </t:template>

   <t:template match="element(my:simpleUserUnion, my:partIntegerUnion)[2]">
      <t:element name="A{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="element(*, my:listUnionType)[1]">
      <t:element name="B{position()}">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="element(*)"> </t:template>
</t:transform>

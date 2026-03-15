<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.example.com/ns/match/var"
   xmlns:t="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="my" version="2.0">
   <!-- Purpose: Test of template matching with "/element($name,$type)" with an explicitly created and 
  				typed element node without a parent inside a variable. -->

   <t:import-schema namespace="http://www.example.com/ns/match/var"
      schema-location="variousTypesSchemaMatch.xsd"/>

   <t:variable name="v">
      <my:simpleUserElem t:validation="strict">123-AB</my:simpleUserElem>
   </t:variable>

   <t:template match="/element(my:simpleUserElem, my:partNumberType)">
      <t:element name="E">
         <t:value-of select="."/>
      </t:element>
   </t:template>
   
   <t:template match="/doc">
      <out>
         <t:apply-templates select="$v"/>
      </out>
   </t:template>   
</t:transform>

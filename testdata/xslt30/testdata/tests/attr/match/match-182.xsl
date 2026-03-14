<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.example.com/ns/match/var"
   xmlns:t="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="my" version="2.0">
   <!-- Purpose: Test of template matching with "//schema-element($name)" with an 
				explicitly created and typed element node with a typed parent.-->

   <t:import-schema namespace="http://www.example.com/ns/match/var"
      schema-location="variousTypesSchemaMatch.xsd"/>

   <t:variable name="var">
      <my:userNode t:validation="strict">
         <my:simpleUserElem>123-AB</my:simpleUserElem>
      </my:userNode>
   </t:variable>

   <t:template match="//schema-element(my:simpleUserElem)">
      <t:element name="E">
         <t:value-of select="."/>
      </t:element>
   </t:template>
   
   <t:template match="/doc">
      <out>
         <t:apply-templates select="$var"/>
      </out>
   </t:template>
   
</t:transform>

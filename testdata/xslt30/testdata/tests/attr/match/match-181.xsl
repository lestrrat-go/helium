<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.example.com/ns/match/var"
   xmlns:t="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="my" version="2.0">
   
   <!-- Test of template matching with schema-element($name) with an explicitly created and 
  				typed element node with a typed parent.-->

   <t:import-schema namespace="http://www.example.com/ns/match/var"
      schema-location="variousTypesSchemaMatch.xsd"/>

   <t:variable name="var">
      <my:userNode t:type="my:userType">
         <my:simpleUserElem>123-AB</my:simpleUserElem>
      </my:userNode>
   </t:variable>

   <t:template match="/">
      <out>
         <t:apply-templates select="$var/my:userNode"/>
      </out>
   </t:template>

   <t:template match="schema-element(my:simpleUserElem)">
      <t:element name="E">
         <t:value-of select="."/>
      </t:element>

   </t:template>

   <t:template match="schema-element(my:userNode)">
      <t:element name="E">
         <t:text>userType was matched</t:text>
      </t:element>
      <t:apply-templates select="*"/>
   </t:template>

   <t:template match="element()">
      <t:text>untyped nodes</t:text>
      <t:text> * </t:text>
      <t:apply-templates select="*"/>
   </t:template>
</t:transform>

<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.example.com/ns/match/var"
   xmlns:t="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="my" version="2.0">
   <!-- Purpose: Test of template matching with element( *,$type) with an explicitly created and typed element 
				node with an untyped parent. Does not match.-->

   <t:import-schema namespace="http://www.example.com/ns/match/var"
      schema-location="variousTypesSchemaMatch.xsd"/>

   <t:variable name="var">
      <my:userNode>
         <my:simpleUserElem t:type="my:partNumberType">123-AB</my:simpleUserElem>
      </my:userNode>
   </t:variable>

   <t:template match="/">
      <out>
         <t:apply-templates select="$var/my:userNode"/>
      </out>
   </t:template>

   <t:template match="element(*, my:partNumberType)">
      <t:element name="E">
         <t:value-of select="."/>
      </t:element>
   </t:template>

   <t:template match="element(*, my:userType)">
      <t:element name="E">
         <t:value-of select="./*[1]"/>
      </t:element>
      <t:apply-templates select="*"/>
   </t:template>

   <t:template match="element()">
      <t:text>untyped nodes</t:text>
      <t:text> * </t:text>
      <t:apply-templates select="*"/>
   </t:template>
</t:transform>

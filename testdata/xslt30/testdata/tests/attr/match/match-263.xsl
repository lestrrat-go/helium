<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
   <!-- Purpose: Test that type annotations are preserved by xsl:mode on-no-match="deep-copy" -->

   <t:import-schema schema-location="match-builtin.xsd"/>
   
   <t:mode on-no-match="deep-copy"/>

   <t:template match="/">
         <t:variable name="temp" as="attribute()">
           <t:apply-templates select="//@attr-boolean-match"/>
         </t:variable>
      <out result="{$temp instance of attribute(attr-boolean-match, xs:boolean)}"/>  
   </t:template>

</t:transform>

<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://example.org/hal"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             xpath-default-namespace="http://www.w3.org/2001/XMLSchema"
             exclude-result-prefixes="my"
             version="2.0">
<!-- Purpose: Test with xpath-default-namespace attribute and Xpath expressions in xsl:use-when. -->

   <t:template match="/my:doc">
      <t:text>
</t:text>
      <out>
         <t:apply-templates select="*"/>
      </out>
   </t:template>

   <t:template match="my:item" use-when="'hello' instance of string">
	     <t:value-of select="'attr1'"/>
      <t:text> * </t:text>
   </t:template>

   <t:template match="my:item" use-when="'hello' instance of decimal">
	     <t:value-of select="'another attr'"/>
      <t:text> * </t:text>
   </t:template>
</t:transform>

<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://uri.test" xmlns:t="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="my" version="2.0">
   <!-- Purpose: Test with xsl:template that has @as="element(*)+" and 
  				returns a sequence of xsl:element or LREs.-->

   <t:template match="/">
      <out>
         <e1>
            <t:call-template name="a1"/>
         </e1>
         <e2>
            <t:call-template name="a2"/>
         </e2>
      </out>
   </t:template>

   <t:template name="a1" as="element(*)+">
      <t:element name="elem">hi1</t:element>
      <t:element name="elem">hi2</t:element>
      <t:element name="item">hi3</t:element>
   </t:template>

   <t:template name="a2" as="element(*)+">
      <my:item>1</my:item>
      <item>2</item>
      <my:item>3</my:item>
      <elem>hello</elem>
   </t:template>
</t:transform>

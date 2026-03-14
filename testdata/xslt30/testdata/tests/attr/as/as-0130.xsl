<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://uri.test" xmlns:t="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="my" version="2.0">
   <!-- Purpose: Test with xsl:template that has @as="element(QName)*" 
  				and returns the empty sequence, a sequence of xsl:element or LREs. -->

   <t:template match="/">
      <out>
         <e1>
            <t:call-template name="a1"/>
         </e1>
         <e2>
            <t:call-template name="a2"/>
         </e2>
         <e3>
            <t:call-template name="a3"/>
         </e3>
      </out>
   </t:template>

   <t:template name="a1" as="element(elem)*">
      <t:sequence select="()"/>
   </t:template>

   <t:template name="a2" as="element(elem)*">
      <t:element name="elem">hi1</t:element>
      <t:element name="elem">hi2</t:element>
      <t:element name="elem">hi3</t:element>
   </t:template>

   <t:template name="a3" as="element(my:item)*">
      <my:item>1</my:item>
      <my:item>2</my:item>
      <my:item>3</my:item>
   </t:template>
</t:transform>

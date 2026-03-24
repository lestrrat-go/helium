<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://uri.test" xmlns:t="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="my" version="2.0">
   <!-- Purpose: Test with xsl:template that has @as="item()" and the sequence constructor contains one of:
				- LRE, xsl:value-of, xsl:sequence with one item, xsl:element, xsl:document-->

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
         <e4>
            <t:call-template name="a4"/>
         </e4>
         <e5>
            <t:call-template name="a5"/>
         </e5>
      </out>
   </t:template>

   <t:template name="a1" as="item()">
      <i1>item</i1>
   </t:template>

   <t:template name="a2" as="item()">
      <t:value-of select="'my string'"/>
   </t:template>

   <t:template name="a3" as="item()">
      <t:sequence select="(2)"/>
   </t:template>

   <t:template name="a4" as="item()">
      <t:element name="elem">hi1</t:element>
   </t:template>

   <t:template name="a5" as="item()">
      <t:document>
         <my:item>1</my:item>
      </t:document>
   </t:template>
</t:transform>

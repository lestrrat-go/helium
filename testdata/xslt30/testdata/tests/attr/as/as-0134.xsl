<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test with xsl:template that has @as="attribute()?" 
  				and returns the empty sequence or an xsl:attribute.-->

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

   <t:template name="a1" as="attribute()?">
      <t:sequence select="()"/>
   </t:template>

   <t:template name="a2" as="attribute()?">
      <t:attribute name="att">1</t:attribute>
   </t:template>
</t:transform>

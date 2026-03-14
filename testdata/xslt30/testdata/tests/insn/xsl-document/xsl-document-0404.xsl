<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test case to verify that nodes in the sequence constructor of xsl:document are copied. -->

   <t:template match="/">
      <t:variable name="y" as="node()">
         <lre/>
      </t:variable>

      <t:variable name="x" as="node()">
         <t:document>
            <t:sequence select="doc,$y"/>
         </t:document>
      </t:variable>

      <out>
         <t:value-of select="doc is $x/doc"/>
         <t:value-of select="$x/lre is $y"/>
      </out>
   </t:template>
</t:transform>

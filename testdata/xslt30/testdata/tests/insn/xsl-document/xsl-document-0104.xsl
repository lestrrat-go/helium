<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test case that uses xsl:copy in the sequence constructor of an xsl:document instruction. -->

   <t:template match="child[2]">
      <t:document>
         <out>
            <t:copy/>
         </out>
      </t:document>
   </t:template>

   <t:template match="child"/>
</t:transform>

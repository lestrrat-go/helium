<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
   <!-- Purpose: Tests a dynamic error evaluating the error code of an xsl:message instruction. -->

   <t:param name="zero" select="0"/>

   <t:template match="/">
      <out>
         <t:message error-code="CODE{100 idiv $zero}">
            <t:text>The messaeg</t:text>
         </t:message>
      </out>
   </t:template>
</t:transform>

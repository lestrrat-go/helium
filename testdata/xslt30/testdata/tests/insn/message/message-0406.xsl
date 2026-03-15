<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
   <!-- Purpose: x. -->

   <t:param name="zero" select="0"/>

   <t:template match="/">
      <out>
         <t:message error-code="{23 + $zero}CODE" terminate="yes">
            <t:text>The message</t:text>
         </t:message>
      </out>
   </t:template>
</t:transform>

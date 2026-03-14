<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Tests a dynamic error evaluating the content of an xsl:message instruction. -->

   <t:param name="zero" select="0"/>

   <t:template match="/">
      <out>
         <t:message>
            <msg>
               <t:value-of select="100 div $zero"/>
            </msg>
         </t:message>
      </out>
   </t:template>
</t:transform>

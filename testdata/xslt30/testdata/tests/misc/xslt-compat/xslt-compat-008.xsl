<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test xsl:fallback inside xsl:sequence. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/">
      <out>
         <t:sequence select="('A','bc','def','ghij','klmno')">
            <t:fallback>
               <t:text>fallback text</t:text>
            </t:fallback>
         </t:sequence>
      </out>
   </t:template>
</t:transform>

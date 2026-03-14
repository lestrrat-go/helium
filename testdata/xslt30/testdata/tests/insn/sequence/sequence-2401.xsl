<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform"
                version="2.0">
<!-- Purpose: xsl:sequence - Content in the sequence constructor is an instruction other than xsl:fallback.-->

   <t:template match="/">
      <out>
         <t:sequence select="doc">
            <foo/>
         </t:sequence>
      </out>
   </t:template>
</t:transform>

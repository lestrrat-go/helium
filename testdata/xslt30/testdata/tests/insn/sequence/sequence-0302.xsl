<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Nest parentheses, with concatenation and range expressions -->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="/">
      <out>
         <t:value-of select="(((1 to 4), ((5,6)), 7 to 9, (10)))" separator=", "/>
      </out>
   </t:template>
</t:transform>

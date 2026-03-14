<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Range expression, upward, positive integer to node holding positive integer -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/">
      <out>
         <t:value-of select="(9 to doc/num)" separator=", "/>
      </out>
   </t:template>
</t:transform>

<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test ceiling() first argument choosing first of multiple nodes. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/" version="1.0">
      <out>
         <t:value-of select="ceiling(doc/b/@x)"/>
      </out>
   </t:template>
</t:transform>

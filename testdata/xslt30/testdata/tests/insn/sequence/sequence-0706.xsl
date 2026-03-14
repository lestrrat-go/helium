<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Range expression, upward, (positive integer) function to (positive integer) function -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/">
      <out>
         <t:value-of select="(count(doc) to string-length('foo'))" separator=", "/>
      </out>
   </t:template>
</t:transform>

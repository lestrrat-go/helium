<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: when both operands of op:concatenate are empty sequences, return empty sequence -->

   <t:template match="/">
      <out>
         <t:text>(</t:text>
         <t:value-of select="for $i in ((),()) return $i" separator=", "/>
         <t:text>)</t:text>
      </out>
   </t:template>
</t:transform>

<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Range expression, node holding positive integer to node holding positive integer, weird name -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/">
      <out>
         <t:apply-templates select="doc"/>
      </out>
   </t:template>

   <t:template match="doc">
      <t:value-of select="(to to to)" separator=", "/>
   </t:template>
</t:transform>

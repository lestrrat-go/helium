<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Use version 1.0 on xsl:attribute to affect (new) select expression. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/" version="1.0">
      <out>
         <t:attribute name="item1" select="substring(DATA/item,1)"/>
      </out>
   </t:template>
</t:transform>

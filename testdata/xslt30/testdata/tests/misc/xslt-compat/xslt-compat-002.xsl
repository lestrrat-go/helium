<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test xsl:fallback inside xsl:next-match. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="DATA">
      <out>
         <t:apply-templates select="item[@name='text']"/>
      </out>
   </t:template>

   <t:template match="item" priority="2">
      <t:text>
</t:text>
      <t:value-of select="."/>
      <t:next-match>
         <t:fallback>
            <t:text> then fallback</t:text>
         </t:fallback>
      </t:next-match>
   </t:template>

   <t:template match="*">
      <t:value-of select="concat(' at ',position())"/>
   </t:template>
</t:transform>

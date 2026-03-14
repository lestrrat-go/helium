<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test the xsl:variable element with missing 'required" name attribute.-->

   <t:variable select="'ABC'"/>

   <t:template match="doc">
      <out> Obvious mistake. </out>
   </t:template>
</t:transform>

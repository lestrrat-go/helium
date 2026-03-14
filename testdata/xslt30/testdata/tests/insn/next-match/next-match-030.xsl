<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
<!-- Purpose: Test xsl:next-match inside xsl:copy with a select attribute-->

   <t:strip-space elements="*"/>

   <t:template match="/">
      <out>
         <t:apply-templates/>
      </out>
   </t:template>

   <t:template match="b" priority="1">
      <x xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:apply-templates/>
      </x>
   </t:template>

   <t:template match="b[@test='c']" priority="2">
      <y xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:copy select="..">
            <t:next-match/>
         </t:copy>
      </y>
   </t:template>

</t:transform>

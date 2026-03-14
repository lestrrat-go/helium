<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test xsl:next-match with one xsl:with-param -->

   <t:template match="/">
      <out>
         <t:apply-templates/>
      </out>
   </t:template>

   <t:template match="b" priority="1">
      <t:param name="para"/>
      <x xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:value-of select="$para"/>
         <t:apply-templates/>
      </x>
   </t:template>

   <t:template match="b[@test='c']" priority="2">
      <y xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:next-match>
            <t:with-param name="para" select="@test"/>
         </t:next-match>
      </y>
   </t:template>
</t:transform>

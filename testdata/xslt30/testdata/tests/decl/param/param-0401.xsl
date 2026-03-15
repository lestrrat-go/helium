<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test to set the value of a variable using xsl:next-match. -->

   <t:template match="doc">
      <out>
         <t:next-match>
            <t:with-param name="par1" select="'hola'"/>
         </t:next-match>
      </out>
   </t:template>

   <t:template match="data">
      <t:variable name="par1" select="'defaultValue'"/>
      <t:value-of select="$par1"/>
   </t:template>

   <t:template match="text()"/>
</t:transform>

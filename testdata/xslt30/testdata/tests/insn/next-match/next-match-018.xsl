<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of xsl:next-match and no other matching templates are found. 
  				Verifies that built-in template rules are invoked. -->

   <t:template match="doc">
      <out>
         <t:text>
</t:text>
         <t:apply-templates select="*">
            <t:with-param name="p1" select="'top'"/>
         </t:apply-templates>
         <t:text>
</t:text>
      </out>
   </t:template>

   <t:template match="tag">
      <tag xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:next-match/>
      </tag>
   </t:template>

   <t:template match="bag">
      <t:text>
</t:text>
	     <bag xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:next-match/>
      </bag>
   </t:template>
</t:transform>

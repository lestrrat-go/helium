<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of xsl:next-match with included stylesheets. -->

   <t:include href="impwparam8.xsl"/>

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
      <t:param name="p1" select="'fallback'"/>
      <main-t xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:value-of select="$p1"/>
      </main-t>
      <t:text>
</t:text>
      <div xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:next-match>
    	       <t:with-param name="p1" select="'primary template'"/>
         </t:next-match>
      </div>
   </t:template>

   <t:template match="bag">
      <t:text>
</t:text>
	     <bag xmlns:xsl="http://www.w3.org/1999/XSL/Transform"/>
   </t:template>
</t:transform>

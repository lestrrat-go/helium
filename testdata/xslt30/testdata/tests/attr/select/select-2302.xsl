<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of select in xsl:for-each when context item is atomic type -->

   <t:template match="doc">
		    <out>
         <t:for-each select="20">
	  			      <t:value-of select="."/>
	  			      <t:text>|</t:text>
			      </t:for-each>
      </out>
	  </t:template>
</t:transform>

<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Tests setting the value of tunnel attribute to an empty string(tunnel="") in <xsl:with-param>. 
       This generates an error." -->

   <t:template match="/">
		    <out>
         <t:apply-templates select="doc">
			         <t:with-param name="par1" select="'foo'" tunnel=""/>
		       </t:apply-templates>
      </out>
	  </t:template>

   <t:template match="doc">
		    <t:param name="par1" select="'xyz'" tunnel="yes"/>
		    <t:value-of select="$par1"/>
	  </t:template>
</t:transform>

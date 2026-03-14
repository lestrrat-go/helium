<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Tests setting the value of tunnel attribute to an empty string(tunnel="") in <xsl:param>. 
                This generated an error.  -->

   <t:template match="/">
		    <out>
         <t:call-template name="temp">
			         <t:with-param name="par1" select="'foo'" tunnel="yes"/>
		       </t:call-template>
      </out>
	  </t:template>

   <t:template name="temp">
		    <t:param name="par1" select="'xyz'" tunnel=""/>
		    <t:value-of select="$par1"/>
	  </t:template>
</t:transform>

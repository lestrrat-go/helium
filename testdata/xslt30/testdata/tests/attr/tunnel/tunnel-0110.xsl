<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: test tunnel params and non-tunnel params within xsl:call-template -->

   <t:template match="/">
		    <out>
         <t:call-template name="temp1">
			         <t:with-param name="par1" select="'foo1'" tunnel="no"/>		
			         <t:with-param name="par2" select="'foo2'" tunnel="yes"/>
			         <t:with-param name="par3" select="'foo3'" tunnel="no"/>			
			         <t:with-param name="par4" select="'foo4'" tunnel="yes"/>			
		       </t:call-template>
      </out>
	  </t:template>

   <t:template name="temp1">
		    <t:param name="par1" select="'xyz1'" tunnel="yes"/>
		    <t:param name="par2" select="'xyz2'" tunnel="yes"/>
		    <t:param name="par3" select="'xyz3'" tunnel="no"/>
		    <t:param name="par4" select="'xyz4'" tunnel="no"/>		
		    <t:value-of select="$par1"/>, <t:value-of select="$par2"/>, <t:value-of select="$par3"/>, <t:value-of select="$par4"/>
	  </t:template>
</t:transform>

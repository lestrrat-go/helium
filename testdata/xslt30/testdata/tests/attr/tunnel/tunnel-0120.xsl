<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
<!-- Purpose: Simple test with a tunnel param (tunnel="1") specified within xsl:call-template
  		but template param has tunnel="0" -->

   <t:template match="/">
		    <out>
         <t:call-template name="temp">
			         <t:with-param name="par1" select="'foo'" tunnel="1"/>
			         <t:with-param name="par2" select="'bar'" tunnel=" 1 "/>
		       </t:call-template>
      </out>
	  </t:template>

   <t:template name="temp">
		    <t:param name="par1" select="'xyz'" tunnel="0"/>
		    <t:param name="par2" select="'zyx'"/>
		    <t:value-of select="$par1"/> and <t:value-of select="$par2"/>
	  </t:template>
</t:transform>

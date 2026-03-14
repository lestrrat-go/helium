<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
	<!-- Purpose: Simple test with a non-tunnel param (tunnel="0") specified within xsl:apply-templates
  		but template param has tunnel="1" -->

   <t:template match="/">
		    <out>
         <t:apply-templates select="doc">
			         <t:with-param name="par1" select="'foo'"/>
			         <t:with-param name="par2" select="'bar'" tunnel="0"/>
		       </t:apply-templates>
      </out>
	  </t:template>

   <t:template match="doc">
		    <t:param name="par1" select="'xyz'" tunnel="1"/>
		    <t:param name="par2" select="'zyx'" tunnel="  1  "/>
		    <t:value-of select="$par1"/> and <t:value-of select="$par2"/>
	  </t:template>
</t:transform>

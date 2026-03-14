<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Simple test with a tunnel param specified within xsl:apply-templates and tunnel="no" -->

   <t:template match="/">
		    <out>
         <t:apply-templates select="doc">
			         <t:with-param name="par1" select="'foo'" tunnel="yes"/>
			         <t:with-param name="par2" select="'bar'" tunnel="yes"/>
		       </t:apply-templates>
      </out>
	  </t:template>

   <t:template match="doc">
		    <t:param name="par1" select="'xyz'" tunnel="no"/>
		    <t:param name="par2" select="'zyx'"/>
		    <t:value-of select="$par1"/> and <t:value-of select="$par2"/>
	  </t:template>
</t:transform>

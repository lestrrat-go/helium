<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to illustrate the accumulation of tunnel parameters from different nesting levels. -->

   <t:template match="/">
  	   <t:text>
</t:text>
		    <out>
         <t:apply-templates select="doc">
			         <t:with-param name="par1" select="'foo'" tunnel="yes"/>
		       </t:apply-templates>
         <t:text>
</t:text>
      </out>
	  </t:template>

   <t:template match="doc">
		    <t:apply-templates select="data">
			      <t:with-param name="par2" select="'boo'" tunnel="yes"/>
		    </t:apply-templates>
	  </t:template>

   <t:template match="data">
		    <t:apply-templates select="inner">
			      <t:with-param name="par3" select="'noo'" tunnel="yes"/>
		    </t:apply-templates>
	  </t:template>

   <t:template match="inner">
		    <t:param name="par1" select="'xyz'" tunnel="yes"/>
		    <t:param name="par2" tunnel="yes"/>
		    <t:param name="par3" select="'hgf'" tunnel="yes"/>
		par1:<t:value-of select="$par1"/>  par2:<t:value-of select="$par2"/>  par3:<t:value-of select="$par3"/>
	  </t:template>

   <t:template match="text()"/>
</t:transform>

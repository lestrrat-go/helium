<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Simple test with a tunnel param specified within xsl:apply-templates
  		and the tunnel parameter will be used after having passed through templates 
  		where it wasn't defined in. -->

   <t:template match="/">
		    <out>
         <t:apply-templates select="doc">
			         <t:with-param name="par1" select="'foo'" tunnel="yes"/>
		       </t:apply-templates>
      </out>
	  </t:template>

   <t:template match="doc">
		    <t:apply-templates/>
	  </t:template>

   <t:template match="data">
		    <t:param name="par1" select="'xyz'" tunnel="yes"/>
		    <t:value-of select="$par1"/>
	  </t:template>

   <t:template match="text()"/>
</t:transform>

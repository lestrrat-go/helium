<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: test tunnel param specified within xsl:apply-templates
  		and another non-tunnel param with the same name should not 
  		override the value -->

   <t:template match="/">
		    <out>
         <t:apply-templates select="doc">
			         <t:with-param name="par1" select="'foo1'" tunnel="yes"/>
			         <t:with-param name="par2" select="'foo2'" tunnel="yes"/>			
		       </t:apply-templates>
      </out>
	  </t:template>

   <t:template match="doc">
		    <t:apply-templates select="data">
			      <t:with-param name="par1" select="'bar1'"/>
			      <t:with-param name="par2" select="'bar2'" tunnel="no"/>			
		    </t:apply-templates>
	  </t:template>

   <t:template match="data">
		    <t:param name="par1" select="'xyz1'" tunnel="yes"/>
		    <t:param name="par2" select="'xyz2'" tunnel="yes"/>		
		    <t:value-of select="$par1"/> and <t:value-of select="$par2"/>
	  </t:template>
</t:transform>

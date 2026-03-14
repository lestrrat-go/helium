<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Tests tunnel parameters specified within xsl:call-templates instruction 
  				with identical names as the global variables. -->

   <t:param name="par1" select="'abc'"/>

   <t:param name="par2" select="'def'"/>

   <t:template match="/">
		    <out>
         <t:call-template name="temp1">
			         <!--<t:with-param name="par1" select="'foo1'" tunnel="no"/>-->
			         <t:with-param name="par2" select="'foo2'" tunnel="yes"/>					
		       </t:call-template>
      </out>
	  </t:template>

   <t:template name="temp1">
		    <t:param name="par1" select="'xyz1'" tunnel="yes"/>
		    <t:param name="par2" select="'zuz'" tunnel="yes"/>
		    <t:value-of select="$par1"/> and <t:value-of select="$par2"/>
	  </t:template>
</t:transform>

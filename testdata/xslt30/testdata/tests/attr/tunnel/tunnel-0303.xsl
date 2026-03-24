<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Tests xsl:param at the stylesheet level with @tunnel="no" 
                The parameter should be ignored and not be tunnelled.-->

   <t:param name="par2" select="'globalVal2'" tunnel="no"/>

   <t:template match="doc">
		    <out>
         <t:next-match>
			         <t:with-param name="par1" select="'abc'" tunnel="yes"/>		
		       </t:next-match>
      </out>
	  </t:template>

   <t:template match="data">
		    <t:param name="par1" select="'123'" tunnel="yes"/>		
		    <t:param name="par2" select="'default'" tunnel="no"/>		
		    <t:value-of select="$par1"/> and <t:value-of select="$par2"/> 
	  </t:template>

   <t:template match="text()"/>
</t:transform>

<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Tests parameter specified within xsl:next-match instruction
       and a global parameter with an identical name as the template parameter, 
       which should be ignored. -->

   <t:param name="par1" select="'global'"/>

   <t:template match="doc">
		    <out>
         <t:next-match>
			         <t:with-param name="par1" select="'abc'"/>		
		       </t:next-match>
      </out>
	  </t:template>

   <t:template match="data">
		    <t:param name="par1" select="'123'"/>		
		    <t:value-of select="$par1"/>
	  </t:template>

   <t:template match="text()"/>
</t:transform>

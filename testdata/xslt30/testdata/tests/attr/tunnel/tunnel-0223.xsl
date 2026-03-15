<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Tests tunnel parameters specified within xsl:next-match with two imported templates.  -->

   <t:import href="import2.xsl"/>

   <t:import href="import4.xsl"/>

   <t:template match="doc">
		    <out>
         <t:next-match>
			         <t:with-param name="par" select="'abc'" tunnel="yes"/>
			         <t:with-param name="par2" select="'dfs'" tunnel="no"/>
			         <t:with-param name="par3" select="'jin'"/>					
		       </t:next-match>
      </out>
	  </t:template>

   <t:template match="data">
		    <t:param name="par" select="'123'" tunnel="yes"/>	
		    <t:param name="par2" select="'456'" tunnel="yes"/>
		    <t:param name="par3" select="'789'" tunnel="yes"/>				
		    <t:value-of select="$par"/> and <t:value-of select="$par2"/> and <t:value-of select="$par3"/>
	  </t:template>

   <t:template match="text()"/>
</t:transform>

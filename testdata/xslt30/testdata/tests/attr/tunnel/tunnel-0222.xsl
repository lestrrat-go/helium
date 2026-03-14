<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Tests tunnel parameters specified within xsl:apply-imports
       with an imported template where the imported template to this stylesheet
       has an imported template itself.  -->

   <t:import href="import4.xsl"/>

   <t:template match="doc">
		    <out>
         <t:apply-imports>
			         <t:with-param name="par" select="'abc'" tunnel="yes"/>
			         <t:with-param name="par2" select="'dfs'" tunnel="no"/>
			         <t:with-param name="par3" select="'jin'"/>			
		       </t:apply-imports>
      </out>
	  </t:template>

   <t:template match="data">
		    <t:param name="par" select="'123'" tunnel="yes"/>		
		    <t:param name="par2" select="'456'" tunnel="yes"/>
		    <t:param name="par3" select="'789'" tunnel="yes"/>				
		    <t:value-of select="$par"/> and <t:value-of select="$par2"/> and <t:value-of select="$par3"/>
	  </t:template>

   <t:template match="node()"/>
</t:transform>

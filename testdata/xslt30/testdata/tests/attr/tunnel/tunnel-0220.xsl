<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Tests tunnel parameter specified within xsl:apply-imports and a 
       global parameter with an identical name as the tunnel parameter, 
       which should be ignored. -->

   <t:import href="import2.xsl"/>

   <t:param name="par1" select="'global'"/>

   <t:template match="doc">
		    <out>
         <t:apply-imports>
			         <t:with-param name="par" select="'abc'" tunnel="yes"/>		
		       </t:apply-imports>
      </out>
	  </t:template>

   <t:template match="data">
		    <t:param name="par" select="'123'" tunnel="yes"/>		
		    <t:value-of select="$par"/>
	  </t:template>

   <t:template match="text()"/>
</t:transform>

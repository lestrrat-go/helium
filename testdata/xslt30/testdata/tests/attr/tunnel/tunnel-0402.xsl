<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to show that if a template with an override calls/applies another, 
  				the effect of the override is not passed to the next level. -->

   <t:template match="/">
		    <t:text>
</t:text>
		    <out>
         <t:call-template name="A">
			         <t:with-param name="X" select="'A1'" tunnel="yes"/>
			         <t:with-param name="Y" select="'A2'" tunnel="yes"/>			
		       </t:call-template>
         <t:text>
</t:text>
      </out>
	  </t:template>

   <t:template name="A">
		    <t:param name="X" select="'defA1'" tunnel="yes"/>
		    <t:param name="Y" select="'defA2'" tunnel="yes"/>
		X inside A: <t:value-of select="$X"/>  and Y inside A: <t:value-of select="$Y"/>		
		    <t:call-template name="B"/>
	  </t:template>

   <t:template name="B">
		    <t:call-template name="C">
			      <t:with-param name="Y" select="'B1'"/>
			      <t:with-param name="Z" select="'B2'"/>		
		    </t:call-template>			
	  </t:template>

   <t:template name="C">
		    <t:param name="Y" select="'C1'"/>
		    <t:param name="Z" select="'C2'"/>
		Y inside C: <t:value-of select="$Y"/> and Z inside C: <t:value-of select="$Z"/>
		    <t:call-template name="D"/>
	  </t:template>

   <t:template name="D">
		    <t:param name="X" select="'D1'" tunnel="yes"/>
		    <t:param name="Y" select="'D2'" tunnel="yes"/>
		X inside D: <t:value-of select="$X"/>  and Y inside D: <t:value-of select="$Y"/>
	  </t:template>
</t:transform>

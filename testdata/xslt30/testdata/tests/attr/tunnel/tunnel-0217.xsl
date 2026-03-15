<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Tests the application of two the tunnel parameters specified 
       within xsl:apply-imports where the latter overrides the value of the 
       first one defined. -->

   <t:import href="import1.xsl"/>

   <t:template match="/">
		    <out>
         <t:apply-imports>
			         <t:with-param name="par" select="'abc'" tunnel="yes"/>		
		       </t:apply-imports>
      </out>
	  </t:template>

   <t:template match="doc">
		    <t:apply-imports>
			      <t:with-param name="par" select="'sos'" tunnel="yes"/>		
		    </t:apply-imports>
	  </t:template>

   <t:template match="data">
		    <t:param name="par" select="'123'" tunnel="yes"/>		
		    <t:value-of select="$par"/>
	  </t:template>

   <t:template match="text()"/>
</t:transform>

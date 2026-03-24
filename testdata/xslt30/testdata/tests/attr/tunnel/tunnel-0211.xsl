<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: test tunnel param overriden by another tunnel param in xsl:next-match -->

   <t:import href="import2.xsl"/>

   <t:template match="doc">
		    <out>
         <t:next-match>
			         <t:with-param name="par" select="'abc'" tunnel="yes"/>		
		       </t:next-match>
      </out>
	  </t:template>

   <t:template match="data">
		    <t:param name="par" select="'123'" tunnel="yes"/>		
		    <t:value-of select="$par"/>
	  </t:template>

   <t:template match="text()"/>
</t:transform>

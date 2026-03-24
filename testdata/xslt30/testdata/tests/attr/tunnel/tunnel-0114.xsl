<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to show that initial template (xsl:template match="/") 
                does not get any tunnel parameters.-->

   <t:template match="/">
		    <t:param name="x" tunnel="yes"/>
		    <out>
         <t:value-of select="$x"/>
      </out>
	  </t:template>
</t:transform>

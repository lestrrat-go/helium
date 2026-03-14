<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case that uses xsl:sequence inside xsl:document -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:variable name="v">
		    <t:value-of select="doc/a"/>
	  </t:variable>

   <t:template match="doc">
  		  <t:document>
  			    <out>
            <t:sequence select="(1,2,3,4,5,6)"/>
            <t:text>     </t:text>
            <t:sequence select="$v"/>
         </out>
  		  </t:document>
	  </t:template>
</t:transform>

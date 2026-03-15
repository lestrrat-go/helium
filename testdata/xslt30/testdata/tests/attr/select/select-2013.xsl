<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of select in xsl:value-of with nodeComp operator -->

   <t:template match="doc">
		    <out>
         <t1>
            <t:value-of select="(if (count(child::*) eq 1) then child::* else child::*[1]) is //*:num1"
                        separator="|"/>
         </t1>
      </out>
	  </t:template>

   <t:template match="text()"/>
</t:transform>

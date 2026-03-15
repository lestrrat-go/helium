<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of select in xsl:for-each with ifExpr -->

   <t:template match="doc">
		    <out>
         <t1>
            <t:for-each select="if (some $temp in ./descendant::* satisfies (string($temp) eq '1')) then name(current()) else 1">
			  	         <t:value-of select="."/>
					 	        <t:text>|</t:text>
		    	     </t:for-each>
         </t1>
      </out>
	  </t:template>

   <t:template match="text()"/>
</t:transform>

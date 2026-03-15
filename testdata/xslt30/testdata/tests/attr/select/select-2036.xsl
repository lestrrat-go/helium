<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of select in xsl:for-each with instanceOf and cast operator -->

   <t:template match="doc">
		    <out>
         <t1>
            <t:for-each select="//attribute() instance of empty-sequence()">
		  	 		       <t:value-of select="."/>
				 	         <t:text>|</t:text>
	    		     </t:for-each>
         </t1>
         <t2>
            <t:for-each select="//attribute()[name() eq 'at5'] instance of empty-sequence()">
					          <t:value-of select="."/>
					          <t:text>|</t:text>
			         </t:for-each>
         </t2>
      </out>
	  </t:template>

   <t:template match="text()"/>
</t:transform>

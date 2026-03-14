<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of select in xsl:for-each with valueComp operator -->

   <xslt:template match="doc">
		    <out>
         <t1>
            <xslt:for-each select="xs:integer((if (count(child::*) eq 1) then child::* else child::*[1])) lt 10">
			  	         <xslt:value-of select="."/>
					 	        <xslt:text>|</xslt:text>
		    	     </xslt:for-each>
         </t1>
      </out>
	  </xslt:template>

   <xslt:template match="text()"/>
</xslt:transform>

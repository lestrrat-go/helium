<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of select in xsl:for-each with processing-instruction(NCName) -->

   <xslt:template match="doc">
		    <out>
         <t1>
            <xslt:for-each select="processing-instruction(pi)">
			  		        <xslt:value-of select="name(.)"/>
			  		        <xslt:text>|</xslt:text>
			  	      </xslt:for-each>
         </t1>
      </out>
	  </xslt:template>

   <xslt:template match="text()"/>
</xslt:transform>

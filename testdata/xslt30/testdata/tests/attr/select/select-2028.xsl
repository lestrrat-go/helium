<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="3.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of select in xsl:for-each with function call: fn:reverse -->

   <xslt:variable name="var"
                  select="(xs:boolean('true'), 2.343, xs:gYearMonth('2005-10'), 3e5)"/>

   <xslt:template match="doc">
		    <out>
         <t1>
            <xslt:for-each select="reverse((sort(../child::element()/attribute()), (xs:boolean('true'), 2.343, xs:gYearMonth('2005-10'), 3e5)))">
		  	          <xslt:value-of select="."/>
				           <xslt:text>|</xslt:text>
	    	      </xslt:for-each>
         </t1>
         <t2>
            <xslt:for-each select="reverse((sort(../child::element()/attribute()), $var )[. castable as xs:integer])">
		  	          <xslt:value-of select="."/>
				           <xslt:text>|</xslt:text>
	     	     </xslt:for-each>
         </t2>
      </out>
	  </xslt:template>

   <xslt:template match="text()"/>
</xslt:transform>

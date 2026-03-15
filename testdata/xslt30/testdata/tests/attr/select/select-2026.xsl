<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of select in xsl:for-each with sequence of atomic values and predicate -->

   <xslt:variable name="var"
                  select="(xs:boolean('true'), 2.343, xs:gYearMonth('2005-10'), 3e5)"/>

   <xslt:template match="doc">
		    <out>
         <t1>
            <xslt:for-each select="../child::element()/attribute()[string(.) ne 'at2']/((xs:boolean('true'), 2.343, xs:gYearMonth('2005-10'), 3e5))[. instance of xs:boolean]">
		  	          <xslt:value-of select="."/>
				 	         <xslt:text>|</xslt:text>
	    	      </xslt:for-each>
         </t1>
         <t2>
            <xslt:for-each select="(../child::element()/attribute()[string(.) ne 'at2'][last()]/$var)[last()-1]">
		  	          <xslt:value-of select="."/>
				 	         <xslt:text>|</xslt:text>
	     	     </xslt:for-each>
         </t2>
      </out>
	  </xslt:template>

   <xslt:template match="text()"/>
</xslt:transform>

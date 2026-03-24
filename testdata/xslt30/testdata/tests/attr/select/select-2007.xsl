<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of select in xsl:value-of with sequence of nodes and atomic values and predicate -->

   <xslt:template match="doc">
		    <out>
         <t1>
            <xslt:value-of select="(../child::element()/attribute()[string(.) ne 'at2'], xs:boolean('true'), 2.343, xs:gYearMonth('2005-10'), 3e5) [position() lt last() and . instance of xs:gYearMonth]"
                           separator="|"/>
         </t1>
      </out>
	  </xslt:template>

   <xslt:template match="text()"/>
</xslt:transform>

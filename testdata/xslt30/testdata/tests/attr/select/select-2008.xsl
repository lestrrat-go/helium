<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of select in xsl:value-of with function call fn:concat -->

   <xslt:template match="doc">
		    <out>
         <t1>
            <xslt:value-of select="concat(xs:gYearMonth('2005-10'), 3e5 )" separator="|"/>
         </t1>
         <t2>
            <xslt:value-of select="*[position() = last()]/concat(name(), 3e5 )" separator="|"/>
         </t2>
      </out>
	  </xslt:template>

   <xslt:template match="text()"/>
</xslt:transform>

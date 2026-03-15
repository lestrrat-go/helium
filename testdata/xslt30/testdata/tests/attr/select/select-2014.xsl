<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of select in xsl:value-of with generalComp operator -->

   <xslt:template match="doc">
		    <out>
         <t1>
            <xslt:value-of select="xs:integer((if (count(child::*) eq 1) then child::* else child::*[1])) = (1 to 4)"
                           separator="|"/>
         </t1>
      </out>
	  </xslt:template>

   <xslt:template match="text()"/>
</xslt:transform>

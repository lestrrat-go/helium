<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test case to verify that in an xsl:value-of instruction zero-length text nodes in the sequence are discarded. -->

   <xslt:template match="doc">
		    <xslt:variable name="zero-length-string" as="xs:string">
  			    <xslt:value-of select="''"/>
		    </xslt:variable>
		    <xslt:variable name="zero-length-text" as="text()">
  			    <xslt:text/>
		    </xslt:variable>
   		<out>
         <xslt:value-of select="('a', '', 'b', $zero-length-string, 'c', $zero-length-text,'d', no-such-node,'e')"
                        separator="|"/>
      </out>
	  </xslt:template>
</xslt:transform>

<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test case to verify that the sequence is atomized inside xsl:comment instruction. -->

   <xslt:template match="/">
   		<out>
         <xslt:text>
</xslt:text>
         <xslt:comment>
   			      <xslt:sequence select="doc/ch, doc//comment(), xs:float('1.234567890123456798'), xs:dayTimeDuration('P020DT03H')"/>
   		    </xslt:comment>
         <xslt:text>
</xslt:text>
      </out>
	  </xslt:template>
</xslt:transform>

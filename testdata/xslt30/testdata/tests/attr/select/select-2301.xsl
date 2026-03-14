<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of select in xsl:value-of for simple content construction -->

   <xslt:variable name="var"> 
		    <xslt:text>t1</xslt:text>
		    <xslt:text>a</xslt:text>
		    <xslt:element name="num1">1</xslt:element>
		    <xslt:element name="num3">3</xslt:element>
		    <xslt:text>t2</xslt:text>
		    <xslt:text/>
		    <xslt:text>t3</xslt:text>
		    <xslt:element name="num4">4</xslt:element>
		    <xslt:element name="num5">5</xslt:element>
		    <xslt:text/>
	  </xslt:variable>

   <xslt:template match="doc">
		    <out>
         <xslt:text>
</xslt:text>
         <a>
            <xslt:value-of select="$var/node()"/>
         </a>
         <xslt:text>
</xslt:text>
         <b>
            <xslt:value-of select="$var/node()" separator="|"/>
         </b>
         <xslt:text>
</xslt:text>
      </out>
	  </xslt:template>
</xslt:transform>

<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test case that uses use-when with an EBV of true on an xsl:call-template instruction. -->

   <xslt:template match="/">
		    <out>
         <xslt:call-template name="temp" use-when="2*2=4">
				        <xslt:with-param name="p1"
                             select="/doc/item[1]/@price"
                             tunnel="yes"
                             use-when="true()"/>
				        <xslt:with-param name="p2"
                             select="/doc/item[2]/@price"
                             tunnel="yes"
                             use-when="false()"/>
				        <xslt:with-param name="p3"
                             select="/doc/item[3]/@price"
                             tunnel="yes"
                             use-when="true()"/>
			      </xslt:call-template>
      </out>
	  </xslt:template>

   <xslt:template name="temp">
		    <xslt:param name="p1" tunnel="yes" select="'def'" use-when="true()"/>
		    <xslt:param name="p2" tunnel="yes" select="'def'"/>
		    <xslt:param name="p3" tunnel="yes" select="'def'" use-when="true()"/>
		    <xslt:sequence select="concat($p1, $p2, $p3)"/>
	  </xslt:template>
</xslt:transform>

<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                >

<!-- PURPOSE: List all the namespace nodes for an element -->

<xsl:template match="/">
<out>
    <xsl:for-each select="//c:chap/namespace::*" xmlns:c="http://chap-default.uri/">
        <xsl:sort select="name()"/>
      <xsl:value-of select="name()"/> = <xsl:value-of select="."/>;
    </xsl:for-each>
</out>
</xsl:template>
 
</xsl:stylesheet>

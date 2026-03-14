<?xml version='1.0'?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:abc="http://abc.uri/">

<!-- Test reference to reserved namespace in strip-space (Saxon bug 4129) -->

    <xsl:strip-space elements="xsl:*"/>
    <xsl:preserve-space elements="xsl:text"/>
    
    <xsl:template name="data">
        <e> <xsl:attribute name="x">
                <xsl:text>  </xsl:text>
            </xsl:attribute> </e>
    </xsl:template>
    
    <xsl:template match="/">
        <out><xsl:value-of select="serialize(//xsl:template[@name='data'])"/></out>
    </xsl:template>
</xsl:stylesheet>

<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

    <xsl:template name="main">
        <xsl:for-each select="1 to 5">
            <xsl:apply-templates select="."/>
        </xsl:for-each>
    </xsl:template>
    
    <xsl:template match=".">
        <out>
            <xsl:apply-templates/>
        </out>
    </xsl:template>

</xsl:stylesheet>

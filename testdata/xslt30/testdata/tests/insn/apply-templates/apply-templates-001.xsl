<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<xsl:template name="main">
    <xsl:for-each select="1 to 5">
        <xsl:apply-templates/>
    </xsl:for-each>
</xsl:template>

</xsl:stylesheet>

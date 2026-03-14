<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

    <!-- PURPOSE: Test copying of namespace nodes -->

    <xsl:template match="/">
        <out>
            <xsl:for-each select="/*/*/*">
                <xsl:copy>PP</xsl:copy>
            </xsl:for-each>
        </out>
    </xsl:template>

</xsl:stylesheet>

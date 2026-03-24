<!-- the simplest possible package: unnamed mode explicitly public, error XTSE0020 -->
<xsl:package version="3.0"
    declared-modes="false"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

    <xsl:mode visibility="final" />
    
    <xsl:template match="." mode="#unnamed">
        <ok><xsl:value-of select="." /></ok>
    </xsl:template>

</xsl:package>   

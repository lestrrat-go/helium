<!-- the simplest possible package: top-level default mode makes a private mode an eligible initial mode -->
<xsl:stylesheet version="3.0"
    default-mode="start"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

    <xsl:mode name="start" visibility="private" />
    
    <xsl:template match="." mode="start">
        <ok><xsl:value-of select="." /></ok>
    </xsl:template>

</xsl:stylesheet>   

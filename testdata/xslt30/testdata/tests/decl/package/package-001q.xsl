<!-- the simplest possible package: top-level default mode makes a private mode an eligible initial mode -->
<xsl:package version="3.0"
    declared-modes="false"
    default-mode="start"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

    <xsl:expose component="mode" names="start" visibility="private" />
    
    <xsl:mode name="start" />
    
    <xsl:template match="." mode="start">
        <ok><xsl:value-of select="." /></ok>
    </xsl:template>

</xsl:package>   

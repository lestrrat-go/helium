<!-- the simplest possible package: xsl:expose has no effect on implicit modes, which are private, initial mode must be public, XTDE0045 -->
<xsl:package version="3.0"
    declared-modes="false"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

    <xsl:expose component="mode" names="*" visibility="public" />
    
    <xsl:template match="." mode="start">
        <ok><xsl:value-of select="." /></ok>
    </xsl:template>

</xsl:package>   

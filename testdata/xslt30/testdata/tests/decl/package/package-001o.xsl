<!-- the simplest possible package: named mode explicitly public, xsl:template with visibility has no influence -->
<xsl:package version="3.0"
    declared-modes="false"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

    <xsl:mode visibility="public" name="start" />
    
    <xsl:template match="." mode="start" name="somename" visibility="private">
        <ok><xsl:value-of select="." /></ok>
    </xsl:template>

</xsl:package>   

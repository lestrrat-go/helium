<!-- the simplest possible package: named mode explicitly private, xsl:template with visibility has no influence, error XTSE0020 -->
<xsl:package version="3.0"
    declared-modes="false"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

    <xsl:mode visibility="private" name="start" />
    
    <xsl:template match="." mode="start" name="somename" visibility="public">
        <not-ok><xsl:value-of select="." /></not-ok>
    </xsl:template>

</xsl:package>   

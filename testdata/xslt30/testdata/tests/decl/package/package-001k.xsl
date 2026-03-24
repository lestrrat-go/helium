<!-- the simplest possible package: unnamed mode explicitly private -->
<xsl:package version="3.0"
    declared-modes="false"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

    <xsl:mode visibility="private" />
    
    <xsl:template match="." mode="start">
        <not-ok />
    </xsl:template>

    <xsl:template match="." mode="#unnamed">
        <ok><xsl:value-of select="." /></ok>
    </xsl:template>

</xsl:package>   

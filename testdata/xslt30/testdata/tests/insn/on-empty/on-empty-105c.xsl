<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
    <!-- basic tests with xsl:on-empty -->
  
    <xsl:template name="xsl:initial-template">
        <empty>
            <xsl:variable name="comments" as="comment()*">
                <xsl:comment select="''"></xsl:comment>
                <xsl:comment select="''"><!-- empty comment --></xsl:comment>
                <xsl:comment></xsl:comment>
            </xsl:variable>
            <xsl:copy-of select="$comments" />
            <xsl:on-empty>
                <this><is><empty/></is></this>
            </xsl:on-empty> 
        </empty>
    </xsl:template>

</xsl:stylesheet>

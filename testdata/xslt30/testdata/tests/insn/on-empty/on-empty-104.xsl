<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
    <!-- basic tests with xsl:on-empty -->
  
    <xsl:template name="xsl:initial-template">
        <empty>
            <xsl:text><!-- significant space --> <!-- is not empty --></xsl:text>
            <xsl:on-empty select="42"/> 
        </empty>
    </xsl:template>

</xsl:stylesheet>

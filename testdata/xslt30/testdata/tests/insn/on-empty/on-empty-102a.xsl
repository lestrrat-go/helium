<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
    <!-- basic tests with xsl:on-empty -->
  
    <xsl:template name="xsl:initial-template">
        <empty>
            <!-- comment is still empty -->
            <xsl:text><!-- empty text node --></xsl:text>
            <xsl:value-of select="(1 to 10)[position() = 11]" />
            <xsl:for-each select="20 to 30"><xsl:text><!--nothing --></xsl:text></xsl:for-each>
            <?pi not a part ?>
            <xsl:on-empty select="for $i in 1 to 100 return $i mod 2"/> 
        </empty>
    </xsl:template>

</xsl:stylesheet>

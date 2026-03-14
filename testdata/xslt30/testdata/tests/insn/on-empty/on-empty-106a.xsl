<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
    <!-- basic tests with xsl:on-empty -->
  
    <xsl:template name="xsl:initial-template">
        <empty>
            <xsl:attribute name="what-is">fourty-two</xsl:attribute>
            <xsl:on-empty select="for $i in 1 to 100 return $i mod 2"/> 
        </empty>
    </xsl:template>

</xsl:stylesheet>

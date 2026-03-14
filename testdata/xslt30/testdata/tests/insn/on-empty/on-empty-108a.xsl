<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
    <!-- basic tests with xsl:on-empty -->

    <xsl:attribute-set name="a">
        <xsl:attribute name="what-is">fourty-two</xsl:attribute>
    </xsl:attribute-set>
  

    <xsl:template name="xsl:initial-template">
        <empty xsl:use-attribute-sets="a">
            <xsl:on-empty select="for $i in 1 to 100 return $i mod 2"/> 
        </empty>
    </xsl:template>

</xsl:stylesheet>

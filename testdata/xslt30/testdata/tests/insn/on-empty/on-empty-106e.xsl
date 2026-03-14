<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
    <!-- basic tests with xsl:on-empty -->
  
    <xsl:template name="xsl:initial-template">
        <empty>
            <xsl:if test="year-from-date(current-date()) gt 1900">
                <xsl:attribute name="what-is">fourty-two</xsl:attribute>
            </xsl:if>
            <xsl:on-empty select="42"/> 
        </empty>
    </xsl:template>

</xsl:stylesheet>

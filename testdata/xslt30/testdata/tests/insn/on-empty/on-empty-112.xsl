<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
    <!-- basic tests with xsl:on-empty in combination with other constructs -->
  
    <xsl:template name="xsl:initial-template">
        <empty> 
            <xsl:for-each select="1 to 100">
                <xsl:value-of select="if(. mod 10) then . else ''" />
                <xsl:on-empty select="'|'"/> 
            </xsl:for-each>
        </empty>
    </xsl:template>

</xsl:stylesheet>

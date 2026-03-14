<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
    <!-- basic tests with xsl:on-empty in combination with other constructs -->
    
    <xsl:template match="/" name="xsl:initial-template">
      <out>
        <xsl:for-each select="1 to 100">
            <xsl:on-non-empty select="''" />
            <xsl:text expand-text="yes">{ if(. = 1) then 'START-->' else ''}</xsl:text>
            <xsl:sequence select="''" />
            <xsl:value-of select="if(. mod 2) then . else ''" />
            <xsl:text expand-text="yes">{''}</xsl:text>
            <xsl:sequence select="if(. = 100) then '&lt;-- END' else ''" />
            <xsl:on-empty select="'|'" />
        </xsl:for-each>
      </out>      
    </xsl:template>

</xsl:stylesheet>

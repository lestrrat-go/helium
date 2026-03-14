<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">
    <!-- basic tests with xsl:on-empty -->
  
    <xsl:template name="xsl:initial-template">
        <empty>            
            <xsl:on-empty select="xs:base64Binary('dkaieKDsiws85eks')"/> 
        </empty>
    </xsl:template>

</xsl:stylesheet>

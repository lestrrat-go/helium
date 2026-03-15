<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">
    <!-- basic tests with xsl:on-empty -->
  
    <xsl:attribute-set name="a">
        <xsl:attribute name="what-is">fourty-two</xsl:attribute>
    </xsl:attribute-set>
  
    <xsl:template name="xsl:initial-template">
        <empty xsl:use-attribute-sets="a">
            <xsl:on-empty select="xs:base64Binary('dkaieKDsiws85eks')"/> 
        </empty>
    </xsl:template>

</xsl:stylesheet>

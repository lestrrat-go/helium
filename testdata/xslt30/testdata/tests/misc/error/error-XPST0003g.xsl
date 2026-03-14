<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: XPath syntax -->
<?spec xpath#errors?><?error XPST0003?>
<!-- processing instruction must be a QName -->

  <xsl:template match="doc">
    <out>
      <xsl:value-of select="processing-instruction(3)"/>
      <xsl:message>Error not detected!</xsl:message>
    </out>
  </xsl:template>

</xsl:stylesheet>
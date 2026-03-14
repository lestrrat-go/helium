<?xml version="1.0" encoding="iso-8859-1"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: XPath syntax -->
<?spec xpath#errors?><?error XPST0003?>

  <xsl:template match="doc">
    <out>
      <xsl:value-of select="7mod2"/> 
      <xsl:message>Error not detected!</xsl:message>
    </out>
  </xsl:template>

</xsl:stylesheet>
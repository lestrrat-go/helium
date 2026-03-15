<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec fo#substring?>
<!-- PURPOSE:   Test of substring where (start < 0) and (start + length < 0). -->
<xsl:template match="/">
  <out>
    <xsl:value-of select="substring(/concepts/@id, string-length(/concepts/@id) - 18, 1)"/>
  </out>
  </xsl:template>
 
</xsl:stylesheet>

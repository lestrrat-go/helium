<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- PURPOSE:   Test of 'string-length()' -->
<?spec fo#func-string-length?>
<xsl:template match="doc">
  <out>
    <xsl:value-of select="string-length()"/> should be 6
  </out>
  </xsl:template>
 
</xsl:stylesheet>

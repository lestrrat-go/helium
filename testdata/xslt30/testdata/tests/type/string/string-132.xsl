<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- PURPOSE:   Test of 'string-length()' with a Unicode surrogate pair -->
<?spec fo#func-string-length?>
<xsl:template match="doc">
  <out>
    <xsl:value-of select="string-length('abc&#x50000;def')"/> should be 7
  </out>
  </xsl:template>
 
</xsl:stylesheet>

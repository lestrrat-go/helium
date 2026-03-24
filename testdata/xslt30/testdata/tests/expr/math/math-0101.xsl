<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-number?>
<!-- PURPOSE:  Test of number() conversion function. -->
<xsl:template match="doc">
  <out>
    <xsl:value-of select="number(n1)"/>
  </out>
  </xsl:template>
 
</xsl:stylesheet>

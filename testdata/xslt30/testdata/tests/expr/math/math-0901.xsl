<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-numeric-mod?>
<!-- PURPOSE:  Test of 'mod' operator. -->
<xsl:template match="doc">
  <out>
    <xsl:value-of select="5 mod 2"/>
  </out>
</xsl:template>
 
</xsl:stylesheet>

<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-starts-with?>
<!-- PURPOSE:   Test of 'starts-with()' function. -->
<xsl:template match="/">
  <out>
    <xsl:value-of select="starts-with('ENCYCLOPEDIA', 'ENCY')"/>
  </out>
</xsl:template>
 
</xsl:stylesheet>

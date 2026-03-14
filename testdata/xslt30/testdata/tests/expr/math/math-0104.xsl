<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <?spec fo#casting-to-numerics?>
<!-- PURPOSE:  Test that the strings "INF", "-INF", and "NaN" are acceptable -->
<xsl:template match="doc">
  <out>
    <xsl:value-of select="number('INF') + 3"/>;
    <xsl:value-of select="number('-INF') + 3"/>;
    <xsl:value-of select="number('NaN') + 3"/>;        
  </out>
  </xsl:template>
 
</xsl:stylesheet>

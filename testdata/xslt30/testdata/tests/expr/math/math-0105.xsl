<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

   <?spec fo#casting-to-numerics?>
<!-- PURPOSE:  Test that the strings "INF", "-INF", and "NaN" are acceptable as floats -->
<xsl:template match="doc" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <out>
    <xsl:value-of select="xs:float('INF') + 3"/>;
    <xsl:value-of select="xs:float('-INF') + 3"/>;
    <xsl:value-of select="xs:float('NaN') + 3"/>;        
  </out>
  </xsl:template>
 
</xsl:stylesheet>

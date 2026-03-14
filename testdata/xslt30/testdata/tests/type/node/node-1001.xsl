<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#axes?>
    <!-- Purpose: Test for value-of with processing-instruction() node test. -->

<xsl:template match="/">
  <out>
    <xsl:apply-templates select="./processing-instruction()"/>	
  </out>
</xsl:template>

<xsl:template match="processing-instruction()">
  <xsl:text>Found-pi...</xsl:text>
  <xsl:value-of select="."/>
</xsl:template>

</xsl:stylesheet>

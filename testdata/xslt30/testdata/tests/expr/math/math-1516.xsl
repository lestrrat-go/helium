<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-numeric-subtract?>
    <!-- Purpose: Test that NaN propagates through subtraction. -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="2 - number('xxx')"/><xsl:text>,</xsl:text>
    <xsl:value-of select="number('xxx') - 10"/>
  </out>
</xsl:template>

</xsl:stylesheet>

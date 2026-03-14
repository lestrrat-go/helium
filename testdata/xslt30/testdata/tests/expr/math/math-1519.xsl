<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-floor?>
<?spec fo#func-ceiling?>
<?spec fo#func-round?>
    <!-- Purpose: Test that NaN propagates through the numeric functions. -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="floor(number('xxx'))"/><xsl:text>,</xsl:text>
    <xsl:value-of select="ceiling(number('xxx'))"/><xsl:text>,</xsl:text>
    <xsl:value-of select="round(number('xxx'))"/>
  </out>
</xsl:template>

</xsl:stylesheet>

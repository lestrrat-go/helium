<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#abbrev?>
    <!-- Purpose: Test that // goes down at least 15 levels. -->

<xsl:template match="doc">
  <out>
    <xsl:for-each select="//title">
      <xsl:value-of select="."/><xsl:text>
</xsl:text>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>

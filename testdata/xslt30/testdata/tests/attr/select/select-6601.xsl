<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#current-function?>

    <!-- Purpose: Simple test of current() in for-each. -->

<xsl:output method="xml" encoding="UTF-8"/>

<xsl:template match="doc">
  <out>
    <xsl:for-each select="foo">
      <xsl:text>&#10;</xsl:text>
      <match>
        <xsl:value-of select="current()/@name"/>
        <xsl:text> = </xsl:text>
        <xsl:value-of select="./@name"/>
      </match>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>

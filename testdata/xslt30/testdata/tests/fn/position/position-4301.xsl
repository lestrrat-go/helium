<?xml version="1.0" encoding="UTF-8"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Try double positional predicates to ensure they at least raise no error -->

<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<xsl:template match="doc">
  <out>
    <xsl:apply-templates select="item[last()][last()]"/>
    <xsl:text>, </xsl:text>
    <xsl:apply-templates select="item[1][last()]"/>
  </out>
</xsl:template>

<xsl:template match="item">
  <xsl:value-of select="."/>
</xsl:template>

</xsl:stylesheet>
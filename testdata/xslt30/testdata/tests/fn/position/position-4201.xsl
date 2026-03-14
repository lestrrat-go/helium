<?xml version="1.0" encoding="UTF-8"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

   <!-- Purpose: Ensure that last() takes on new value when context changes -->

<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<xsl:template match="doc">
  <out>
    <xsl:apply-templates select="item[last()-1]"/>
    <xsl:text>, </xsl:text>
    <xsl:apply-templates select="item[last()]"/>
  </out>
</xsl:template>

<xsl:template match="item">
  <xsl:value-of select="last()"/><!-- should be 1 -->
</xsl:template>

</xsl:stylesheet>
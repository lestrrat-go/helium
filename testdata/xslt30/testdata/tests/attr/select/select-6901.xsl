<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Purpose: Try absolute path when current node is in middle of tree -->

<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<xsl:template match="/">
  <out>
    <xsl:apply-templates select="/para/font[@color='green']" />
  </out>
</xsl:template>

<xsl:template match="font">
  <rel><xsl:apply-templates select="..//text()" /></rel>
  <abs><xsl:apply-templates select="//text()" /></abs>
</xsl:template>

</xsl:stylesheet>
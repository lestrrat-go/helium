<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-path-expressions?>
    <!-- Purpose: Compound test for following-sibling:: and preceding-sibling:: with explicit iteration. -->

<xsl:template match="/">
  <out>
    <xsl:for-each select="//center">
      <xsl:for-each select="following-sibling::*"><xsl:text>
From </xsl:text><xsl:value-of select="name(.)"/><xsl:text>:</xsl:text>
        <xsl:apply-templates select="preceding-sibling::*"/>
      </xsl:for-each>
    </xsl:for-each>
  </out>
</xsl:template>

<xsl:template match="*">
  <xsl:text> </xsl:text><xsl:value-of select="name(.)"/>
</xsl:template>

</xsl:stylesheet>

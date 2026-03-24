<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-path-expressions?>
    <!-- Purpose: Test for '..//name' and an attribute. -->

<xsl:template match="/docs">
  <out>
    <xsl:for-each select="doc">
      <xsl:apply-templates select="..//foo"/>
    </xsl:for-each>
  </out>
</xsl:template>

<xsl:template match="*">
  <xsl:value-of select="@att1"/>
</xsl:template>

</xsl:stylesheet>
<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-path-expressions?>
    <!-- Purpose: Test for '@' to select an attribute, with name of attribute. -->

<xsl:template match="/">
  <out>
    <xsl:for-each select="//center">
      <xsl:apply-templates select="@center-attr-2"/>
    </xsl:for-each>
  </out>
</xsl:template>

<xsl:template match="*|@*">
  <xsl:value-of select="name(.)"/>
</xsl:template>

</xsl:stylesheet>

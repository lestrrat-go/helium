<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#built-in-rule?>
<!-- tests that parameters to a built-in template are passed through -->

<xsl:template match="top">
  <xsl:variable name="x">
    <inner/>
  </xsl:variable>
    <xsl:apply-templates select="$x" mode="m">
        <xsl:with-param name="hi" select="42"/>
    </xsl:apply-templates>
</xsl:template>

<xsl:template match="/" mode="m">
  <xsl:param name="hi" select="22"/>
  <xsl:apply-imports>
    <xsl:with-param name="hi" select="$hi"/>
  </xsl:apply-imports>
</xsl:template>

<xsl:template match="inner" mode="m">
  <xsl:param name="hi" select="21"/>
  <z><xsl:value-of select="$hi"/></z>
</xsl:template>

</xsl:stylesheet>
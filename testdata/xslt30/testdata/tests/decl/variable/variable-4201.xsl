<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#with-param?>
    <!-- Purpose: Test for passing numeric param via apply-templates, with sorting -->

<xsl:output method="xml" encoding="UTF-8"/>

<xsl:template match="doc">
  <out>
    <xsl:apply-templates select="a-set">
      <xsl:sort select="@s" data-type="number" order="ascending"/>
      <xsl:with-param name="total" select="sum(a-set/a)"/>
    </xsl:apply-templates>
  </out>
</xsl:template>

<xsl:template match="a-set">
  <xsl:param name="total">Default text in param 1</xsl:param>
  <xsl:text>&#10;</xsl:text>
  <list from="{@g}">
    <xsl:attribute name="proportion">
      <xsl:value-of select="concat(sum(./a/text()),'/',$total)"/>
    </xsl:attribute>
    <xsl:for-each select="./a/text()">
      <xsl:value-of select="."/>
      <xsl:text>,</xsl:text>
    </xsl:for-each>
  </list>
</xsl:template>

</xsl:stylesheet>
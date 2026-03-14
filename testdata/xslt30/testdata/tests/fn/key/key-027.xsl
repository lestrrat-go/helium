<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#keys?>
    <!-- Purpose: Use key() for sorting with apply-templates. -->

<xsl:key name="MonthNum" match="monthtab/entry/number" use="../name" />

<xsl:template match="doc">
  <out>
    <xsl:text>Birthdays in chronological order...
</xsl:text>
    <xsl:apply-templates select="birthday">
      <xsl:sort select="key('MonthNum',month)" data-type="number" />
      <xsl:sort select="day" data-type="number" />
    </xsl:apply-templates>
  </out>
</xsl:template>

<xsl:template match="birthday">
  <xsl:value-of select="@person"/><xsl:text>: </xsl:text>
  <xsl:value-of select="month"/><xsl:text> </xsl:text>
  <xsl:value-of select="day"/><xsl:text>
</xsl:text>
</xsl:template>

</xsl:stylesheet>

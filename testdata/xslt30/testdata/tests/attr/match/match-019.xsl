<?xml version="1.0"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<?spec xslt#patterns?>
    <!-- Purpose: Test step//step[predicate], with positional predicate, to show
     that position numbering applies "relative to the child axis", not //. -->

<xsl:output method="xml" encoding="utf-8"/>

<xsl:template match="/">
  <out>
    <xsl:for-each select="*">
      <xsl:apply-templates/>
    </xsl:for-each>
  </out>
</xsl:template>

<xsl:template match="chapter//footnote[1]">
  <xsl:text>
</xsl:text>
  <first>
    <xsl:value-of select="."/>
  </first>
</xsl:template>

<xsl:template match="chapter//footnote[position() != 1]">
  <other/>
</xsl:template>

<xsl:template match="text()"/><!-- Suppress text matching -->

</xsl:stylesheet>



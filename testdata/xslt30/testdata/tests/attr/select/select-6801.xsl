<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-path-expressions?>
    <!-- Purpose: Try a positional predicate on descendants of a node-set variable -->

<xsl:output method="xml" encoding="UTF-8"/>

<!-- Collect a node-set, outside any template -->
<xsl:variable name="all" select="//OL[@real='yes']"/>

<xsl:template match="/">
  <out>
    <xsl:apply-templates select="doc/OL//tag"/>
  </out>
</xsl:template>

<xsl:template match="tag">
  <xsl:text>Found the tag...
</xsl:text>
  <xsl:apply-templates select="$all/LI[@flag][last()]"/><xsl:text>
</xsl:text>
</xsl:template>

<xsl:template match="LI[@flag]">
  <xsl:value-of select="."/><xsl:text>,</xsl:text>
</xsl:template>

</xsl:stylesheet>
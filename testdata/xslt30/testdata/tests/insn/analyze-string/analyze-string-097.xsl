<?xml version='1.0'?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0" xmlns:e="http://example.org">
 <xsl:variable name="pat1">("[^"]*")|(\{[^}]+})|(,)|([^=\-+*/();:,.$&lt;>^!]+(?:\.[^=\-+*/();:,.$&lt;>^!]+)*\()|([)])|(^=|\()|((?:(?:'[^']+')|(?:\[[0-9]+\][^!]*)|(?:[a-zA-Z_][a-zA-Z0-9._]*)!))|(\$?[A-Z]+\$?[0-9]+)|([a-zA-Z_\\][a-zA-Z0-9._]*)|(.)</xsl:variable>
 <xsl:param name="pat" select="$pat1"/><!-- xsl:param for refinement debugging by passing in the pattern -->
 <xsl:param name="formula">SUM(A1:B2)</xsl:param>

 <xsl:template match="/" name="main">
  <xsl:copy-of select="analyze-string($formula,$pat)"/>
 </xsl:template>
</xsl:stylesheet>

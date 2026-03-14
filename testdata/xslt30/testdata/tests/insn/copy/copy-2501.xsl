<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#shallow-copy?>
    <!-- Purpose: Make an RTF of the whole doc and watch for incorrect xml-decl placement. -->

<xsl:template match="/">
  <xsl:variable name="var1">
    <xsl:apply-templates/>
  </xsl:variable>
  <out>
    <xsl:copy-of select="$var1"/>
  </out>
</xsl:template>

<xsl:template match="*|@*|comment()|processing-instruction()|text()">
  <xsl:copy>
    <xsl:apply-templates select="*|@*|comment()|processing-instruction()|text()"/>
  </xsl:copy>
</xsl:template>

</xsl:stylesheet>
<?xml version="1.0" ?>
<xsl:stylesheet version='2.0' xmlns:xsl='http://www.w3.org/1999/XSL/Transform'>

  <!-- Purpose: Look for bug in building union for identity transform. -->

<xsl:template match="/lot">
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="*|@*|comment()|processing-instruction()|text()">
  <xsl:copy>
    <xsl:apply-templates select="*|@*|comment()|processing-instruction()|text()"/>
  </xsl:copy>
</xsl:template>

</xsl:stylesheet>
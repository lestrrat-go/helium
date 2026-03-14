<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#shallow-copy?>
    <!-- Purpose: Make each comment be the current node, and copy it. -->

<xsl:template match="/">
  <out>
    <xsl:for-each select="//comment()">
      <xsl:copy/><xsl:text>
</xsl:text>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
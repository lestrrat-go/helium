<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#lexical-structure?>
    <!-- Purpose: NCName followed by ( must be recognized as a NodeType
      or FunctionName. -->

<xsl:template match="doc">
  <out>
    <xsl:for-each select="comment()">
      <xsl:copy/>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
